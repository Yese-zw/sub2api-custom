<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ localText("App 更新", "App update") }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{
              localText(
                "桌面 App 每 30 分钟从 /api/v1/app/update/{平台} 读取这里发布的版本；高于本机版本时会在后台下载，并提示用户重启更新。",
                "The desktop App reads the release published here from /api/v1/app/update/{target} every 30 minutes; when it is newer than the installed version, it downloads in the background and asks the user to restart.",
              )
            }}
          </p>
        </div>
        <div class="flex items-center gap-3">
          <span class="text-sm text-gray-600 dark:text-gray-300">
            {{ form.enabled ? localText("已发布", "Published") : localText("未发布", "Not published") }}
          </span>
          <Toggle v-model="form.enabled" :disabled="loading" />
        </div>
      </div>
    </div>

    <div v-if="loading" class="p-6 text-sm text-gray-500 dark:text-gray-400">
      {{ localText("加载中…", "Loading…") }}
    </div>
    <div v-else class="space-y-6 p-6">
      <div class="grid gap-6 md:grid-cols-2">
        <div>
          <label for="app-update-version" class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ localText("版本号", "Version") }}
          </label>
          <input
            id="app-update-version"
            v-model="form.version"
            type="text"
            class="input font-mono"
            placeholder="0.5.5"
          />
          <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
            {{ localText("必须与安装包内的版本一致，App 只会升级到更高版本。", "Must match the version inside the packages; the App only upgrades to newer versions.") }}
          </p>
        </div>
        <div>
          <label for="app-update-pub-date" class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ localText("发布时间", "Publish date") }}
          </label>
          <input
            id="app-update-pub-date"
            v-model="form.pub_date"
            type="text"
            class="input font-mono"
            placeholder="2026-10-28T12:00:00Z"
          />
          <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
            {{ localText("RFC 3339 格式；发布时留空则使用当前时间。", "RFC 3339; left empty, the current time is used when published.") }}
          </p>
        </div>
      </div>

      <div>
        <label for="app-update-notes" class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ localText("更新说明", "Release notes") }}
        </label>
        <textarea
          id="app-update-notes"
          v-model="form.notes"
          rows="5"
          maxlength="20000"
          class="input"
          :placeholder="localText('本次更新的内容', 'What changed in this release')"
        />
      </div>

      <div class="space-y-4">
        <div>
          <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ localText("安装包", "Packages") }}
          </h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{
              localText(
                "填写 Tauri updater 产物（macOS 为 .app.tar.gz，Windows 为 NSIS 安装包）的下载地址，以及同名 .sig 文件的完整内容。签名必须由 App 内置公钥对应的私钥生成。不填的平台不会收到更新。",
                "Enter the download URL of each Tauri updater artifact (.app.tar.gz on macOS, the NSIS installer on Windows) and the full contents of its .sig file. Signatures must come from the private key matching the App's built-in public key. Platforms left empty receive no update.",
              )
            }}
          </p>
        </div>
        <div
          v-for="target in targets"
          :key="target"
          class="rounded-lg border border-gray-200 p-4 dark:border-dark-600"
        >
          <div class="mb-3 flex items-center justify-between">
            <span class="font-mono text-sm font-medium text-gray-900 dark:text-white">{{ target }}</span>
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ targetLabel(target) }}</span>
          </div>
          <div class="space-y-3">
            <input
              v-model="platformFor(target).url"
              type="url"
              class="input font-mono text-xs"
              :aria-label="`${target} URL`"
              placeholder="https://cdn.example.com/releases/v0.5.5/LSAGENT_aarch64.app.tar.gz"
            />
            <textarea
              v-model="platformFor(target).signature"
              rows="2"
              class="input font-mono text-xs"
              :aria-label="`${target} signature`"
              :placeholder="localText('.sig 文件内容', '.sig file contents')"
            />
          </div>
        </div>
      </div>

      <div class="flex justify-end">
        <button type="button" class="btn btn-primary" :disabled="saving" @click="save">
          {{ saving ? localText("保存中…", "Saving…") : localText("保存 App 更新", "Save App update") }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { adminAPI } from "@/api";
import type { AppUpdatePlatform, AppUpdateSettings } from "@/api/admin/settings";
import Toggle from "@/components/common/Toggle.vue";
import { useAppStore } from "@/stores";
import { extractApiErrorMessage } from "@/utils/apiError";

const { locale, t } = useI18n();
const appStore = useAppStore();
const isZhLocale = computed(() => locale.value.startsWith("zh"));

function localText(zh: string, en: string): string {
  return isZhLocale.value ? zh : en;
}

const TARGET_LABELS: Record<string, [string, string]> = {
  "darwin-aarch64": ["macOS Apple 芯片", "macOS Apple silicon"],
  "darwin-x86_64": ["macOS Intel", "macOS Intel"],
  "windows-x86_64": ["Windows x64", "Windows x64"],
  "windows-aarch64": ["Windows ARM64", "Windows ARM64"],
};

const loading = ref(true);
const saving = ref(false);
const targets = ref<string[]>(Object.keys(TARGET_LABELS));
const form = reactive<AppUpdateSettings>({
  enabled: false,
  version: "",
  notes: "",
  pub_date: "",
  platforms: {},
});

function targetLabel(target: string): string {
  const label = TARGET_LABELS[target];
  return label ? localText(label[0], label[1]) : "";
}

function platformFor(target: string): AppUpdatePlatform {
  if (!form.platforms[target]) {
    form.platforms[target] = { url: "", signature: "" };
  }
  return form.platforms[target];
}

function apply(settings: AppUpdateSettings, serverTargets: string[]) {
  if (serverTargets.length > 0) targets.value = serverTargets;
  form.enabled = settings.enabled;
  form.version = settings.version;
  form.notes = settings.notes;
  form.pub_date = settings.pub_date;
  form.platforms = {};
  for (const target of targets.value) {
    const platform = settings.platforms?.[target];
    form.platforms[target] = { url: platform?.url ?? "", signature: platform?.signature ?? "" };
  }
}

async function load() {
  loading.value = true;
  try {
    const data = await adminAPI.settings.getAppUpdateSettings();
    apply(data.settings, data.targets);
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t("common.error")));
  } finally {
    loading.value = false;
  }
}

async function save() {
  saving.value = true;
  try {
    const data = await adminAPI.settings.updateAppUpdateSettings({
      ...form,
      platforms: { ...form.platforms },
    });
    apply(data.settings, data.targets);
    appStore.showSuccess(localText("App 更新已保存", "App update saved"));
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t("common.error")));
  } finally {
    saving.value = false;
  }
}

onMounted(load);
</script>
