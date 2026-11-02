package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// 鹈鹕 / 自由创作测试（本地）：让模型生成一个单文件 HTML 动画，提取 HTML 后按规则初步判定质量。
// 判定只基于可量化的结构特征（图形元素数量、是否包含动画、体积），用于发现明显的降智表现
// （输出空洞、缺失动画等），不是对画面好坏的最终评价，结果仅供人工对比。
const (
	iqPelicanPrompt = "Generate a single self-contained HTML file with an inline SVG animation of a pelican riding a bicycle. " +
		"Draw the pelican (beak, pouch, wing, legs) and the bicycle (two wheels, frame, pedals) with many SVG shapes, " +
		"and animate it with CSS keyframes or SMIL so the wheels spin and the pelican pedals. " +
		"No external resources or libraries. Reply with ONLY the HTML inside one ```html code block."

	iqCreativePrompt = "Create a delightful, self-contained single-file HTML animation of anything you like. " +
		"Use inline SVG or canvas, with plenty of detail and smooth motion. " +
		"No external resources or libraries. Reply with ONLY the HTML inside one ```html code block."

	// 判定阈值
	iqPelicanMinBytes      = 800 // 低于此体积视为过于简陋
	iqPelicanMinShapes     = 8   // 低于此图形数视为过于简陋
	iqPelicanGoodShapes    = 15  // 达到此图形数且含动画视为正常
	iqPelicanQualityNormal = "normal"
)

var (
	iqHTMLFence     = regexp.MustCompile("(?is)```(?:html|svg|xml)?\\s*\\n(.*?)```")
	iqHTMLDocStart  = regexp.MustCompile(`(?i)<!doctype\s+html|<html[\s>]|<svg[\s>]`)
	iqHTMLDocEnd    = regexp.MustCompile(`(?i)</html>|</svg>`)
	iqShapeTag      = regexp.MustCompile(`(?i)<(path|circle|ellipse|rect|polygon|polyline|line)\b`)
	iqAnimationHint = regexp.MustCompile(`(?i)<animate\w*\b|@keyframes|requestAnimationFrame|setInterval|animation\s*:|<canvas\b`)
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

// assessIQHTML 按结构特征给出 quality（normal / suspicious / degraded）与判定依据。
func assessIQHTML(html string) (quality, reason string) {
	shapes := len(iqShapeTag.FindAllString(html, -1))
	animated := iqAnimationHint.MatchString(html)
	size := len(html)
	facts := fmt.Sprintf("图形元素 %d 个，体积 %d 字节，%s动画", shapes, size, map[bool]string{true: "含", false: "无"}[animated])
	switch {
	case shapes < iqPelicanMinShapes || size < iqPelicanMinBytes:
		return "degraded", "输出过于简陋：" + facts
	case shapes >= iqPelicanGoodShapes && animated:
		return iqPelicanQualityNormal, "结构完整：" + facts
	case !animated:
		return "suspicious", "缺少动画：" + facts
	default:
		return "suspicious", "细节偏少：" + facts
	}
}

// runPelican 对单个模型运行鹈鹕（creative=true 时为自由创作）测试。
func (s *ChannelMonitorIQService) runPelican(ctx context.Context, m *ChannelMonitor, model string, creative bool) (*ChannelMonitorIQResult, error) {
	benchmark, prompt := IQBenchmarkPelican, iqPelicanPrompt
	if creative {
		benchmark, prompt = IQBenchmarkCreative, iqCreativePrompt
	}
	text, started, finished, err := s.callUpstream(ctx, m, model, prompt, iqPelicanTimeout, iqPelicanMaxTokens)
	if err != nil {
		return nil, err
	}
	html := extractIQHTML(text)
	if html == "" {
		return nil, errors.New("no html found in the answer")
	}
	quality, reason := assessIQHTML(html)
	return &ChannelMonitorIQResult{
		MonitorID: m.ID, Model: model, Benchmark: benchmark, Quality: quality, Reason: reason, HTML: html,
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
