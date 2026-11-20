package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"

	judgerassets "github.com/Wei-Shaw/sub2api/internal/pkg/htmljudger/assets"
	"github.com/Wei-Shaw/sub2api/internal/pkg/htmljudger/judge"
)

// 鹈鹕 / 自由创作测试：让模型生成 HTML 动画，再用 html-judger 的冻结 v6 判定器打分。
// 鹈鹕测试的提示词必须与题库原文逐字节一致，才能走同题参照；每次轮换题库中的一道。
// 自由创作使用固定的题库外提示词，判定退回原始阈值。判定看的是写码风格，不是画面好坏。
const iqCreativePrompt = "Create a delightful, self-contained single-file HTML animation of anything you like. " +
	"Use inline SVG or canvas, with plenty of detail and smooth motion. " +
	"No external resources or libraries. Reply with ONLY the HTML inside one ```html code block."

var (
	iqHTMLJudgerOnce sync.Once
	iqHTMLJudger     *judge.Judger
	iqHTMLJudgerErr  error
)

var (
	iqHTMLFence    = regexp.MustCompile("(?is)```(?:html|svg|xml)?\\s*\\n(.*?)```")
	iqHTMLDocStart = regexp.MustCompile(`(?i)<!doctype\s+html|<html[\s>]|<svg[\s>]`)
	iqHTMLDocEnd   = regexp.MustCompile(`(?i)</html>|</svg>`)
)

// extractIQHTML 从模型回答里提取 HTML：优先取代码块，其次取 <!doctype/<html/<svg 到结尾标签。
func extractIQHTML(text string) string {
	for _, m := range iqHTMLFence.FindAllStringSubmatch(text, -1) {
		body := strings.TrimSpace(m[1])
		if iqHTMLDocStart.MatchString(body) {
			return body
		}
	}
	loc := iqHTMLDocStart.FindStringIndex(text)
	if loc == nil {
		return ""
	}
	rest := text[loc[0]:]
	ends := iqHTMLDocEnd.FindAllStringIndex(rest, -1)
	if len(ends) == 0 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:ends[len(ends)-1][1]])
}

func loadIQHTMLJudger() (*judge.Judger, error) {
	iqHTMLJudgerOnce.Do(func() {
		iqHTMLJudger, iqHTMLJudgerErr = judge.Load(judgerassets.FS)
	})
	return iqHTMLJudger, iqHTMLJudgerErr
}

// iqNextBankPrompt 按上一题在题库中的位置取下一题。last 为空时从第一题开始。
// 结果表每个（监控, 模型, 检测类型）只保留最近 iqMaxPerModel 条，不能用现存行数当序号：
// 窗口填满后行数不再增加，题号会停住，之后每次都是同一题。
func iqNextBankPrompt(prompts []string, last string) (string, error) {
	if len(prompts) == 0 {
		return "", errors.New("html judge prompt bank is empty")
	}
	if last == "" {
		return prompts[0], nil
	}
	for i, p := range prompts {
		if p == last {
			return prompts[(i+1)%len(prompts)], nil
		}
	}
	// 上一题已不在题库（题库改过）。回到第一题；这次结果写入后，下次会从它往后轮换。
	return prompts[0], nil
}

func (s *ChannelMonitorIQService) nextPelicanPrompt(ctx context.Context, monitorID int64, model string) (string, error) {
	j, err := loadIQHTMLJudger()
	if err != nil {
		return "", fmt.Errorf("html judge unavailable: %w", err)
	}
	var last string
	// 取该模型最近一次鹈鹕结果里保存的原题。没有记录时 last 为空。
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE((
			SELECT label FROM channel_monitor_iq_results
			WHERE monitor_id = $1 AND model = $2 AND benchmark = $3
			ORDER BY created_at DESC, id DESC
			LIMIT 1
		), '')`,
		monitorID, model, IQBenchmarkPelican).Scan(&last); err != nil {
		return "", err
	}
	return iqNextBankPrompt(j.BankPrompts(), last)
}

func formatIQJudgeReason(r judge.Result) string {
	if r.Score != nil {
		return fmt.Sprintf("%s · %s · z=%.3f · margin %.3f", r.Class(), r.Method, *r.Score, r.Margin)
	}
	return fmt.Sprintf("%s · %s · margin %.3f", r.Class(), r.Method, r.Margin)
}

// judgeIQHTML 用 html-judger 判定。quality 为 normal / degraded / borderline。
func judgeIQHTML(prompt, html string) (quality, reason string, err error) {
	j, err := loadIQHTMLJudger()
	if err != nil {
		return "", "", fmt.Errorf("html judge unavailable: %w", err)
	}
	r := j.Judge(prompt, html)
	return r.Class(), formatIQJudgeReason(r), nil
}

// runPelican 对单个模型运行鹈鹕（creative=true 时为自由创作）测试。
func (s *ChannelMonitorIQService) runPelican(ctx context.Context, m *ChannelMonitor, model string, creative bool) (*ChannelMonitorIQResult, error) {
	benchmark, prompt := IQBenchmarkPelican, ""
	var err error
	if creative {
		benchmark, prompt = IQBenchmarkCreative, iqCreativePrompt
	} else if prompt, err = s.nextPelicanPrompt(ctx, m.ID, model); err != nil {
		return nil, err
	}
	text, started, finished, err := s.callUpstream(ctx, m, model, prompt, iqPelicanTimeout, iqPelicanMaxTokens)
	if err != nil {
		return nil, err
	}
	html := extractIQHTML(text)
	if html == "" {
		return nil, errors.New("no html found in the answer")
	}
	quality, reason, err := judgeIQHTML(prompt, html)
	if err != nil {
		return nil, err
	}
	return &ChannelMonitorIQResult{
		MonitorID: m.ID, Model: model, Benchmark: benchmark, Quality: quality, Label: prompt, Reason: reason, HTML: html,
		DurationMs: finished.Sub(started).Milliseconds(), StartedAt: started, FinishedAt: finished,
	}, nil
}

// runPelicanModel 运行鹈鹕测试并保存结果，失败原因记入内存供弹窗展示。
// creative 是鹈鹕测试的配置：开启后只请求自由创作，不再请求鹈鹕。
func (s *ChannelMonitorIQService) runPelicanModel(ctx context.Context, m *ChannelMonitor, model string, creative bool) {
	bm := IQBenchmarkPelican
	if creative {
		bm = IQBenchmarkCreative
	}
	r, err := s.runPelican(ctx, m, model, creative)
	if err != nil {
		slog.Warn("channel_monitor_iq: pelican failed", "monitor_id", m.ID, "model", model, "benchmark", bm, "error", err)
		s.recordFailure(m.ID, model, bm, err)
		return
	}
	if err := s.saveResult(ctx, r); err != nil {
		slog.Warn("channel_monitor_iq: save pelican failed", "monitor_id", m.ID, "benchmark", bm, "error", err)
		return
	}
	s.clearFailure(m.ID, model, bm)
}
