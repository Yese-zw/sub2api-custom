<template>
  <div class="border-t border-gray-200 pt-4 mt-4 dark:border-dark-400">
    <div class="mb-3 flex items-start justify-between gap-3">
      <div class="min-w-0 flex-1">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t("admin.groups.uaBinding.title") }}
        </label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t("admin.groups.uaBinding.hint") }}
        </p>
      </div>
      <Toggle
        data-testid="ua-binding-toggle"
        :aria-label="t('admin.groups.uaBinding.enable')"
        v-model="enabled"
      />
    </div>

    <p v-if="!enabled" class="text-xs text-gray-500 dark:text-gray-400">
      {{ t("admin.groups.uaBinding.disabledHint") }}
    </p>

    <div v-else ref="containerRef" class="space-y-3">
      <p class="text-xs text-amber-600 dark:text-amber-400">
        {{ t("admin.groups.uaBinding.enabledHint") }}
      </p>

      <div
        v-for="rule in rules"
        :key="rule.key"
        class="rounded-lg border border-gray-200 p-3 dark:border-dark-600"
        data-testid="ua-binding-rule"
      >
        <div class="flex items-start gap-3">
          <div class="min-w-0 flex-1 space-y-2">
            <div>
              <label class="input-label text-xs">{{
                t("admin.groups.uaBinding.ruleName")
              }}</label>
              <input
                v-model="rule.name"
                type="text"
                class="input text-sm"
                maxlength="100"
                :placeholder="t('admin.groups.uaBinding.ruleNamePlaceholder')"
              />
            </div>
            <div>
              <label class="input-label text-xs">{{
                t("admin.groups.uaBinding.patterns")
              }}</label>
              <textarea
                v-model="rule.patternsText"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('admin.groups.uaBinding.patternsPlaceholder')"
              />
              <p class="mt-1 text-xs text-gray-400">
                {{ t("admin.groups.uaBinding.patternsHint") }}
              </p>
            </div>
            <div>
              <label class="input-label text-xs">{{
                t("admin.groups.uaBinding.accounts")
              }}</label>
              <div
                v-if="rule.accountIds.length > 0"
                class="mb-2 flex flex-wrap gap-1.5"
              >
                <span
                  v-for="id in rule.accountIds"
                  :key="id"
                  class="inline-flex items-center gap-1 rounded-full bg-primary-100 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
                >
                  {{ accountLabel(id) }}
                  <button
                    type="button"
                    class="ml-0.5 text-primary-500 hover:text-primary-700 dark:hover:text-primary-200"
                    :aria-label="`remove account ${id}`"
                    @click="removeAccount(rule, id)"
                  >
                    <Icon name="x" size="xs" />
                  </button>
                </span>
              </div>
              <div class="relative">
                <input
                  v-model="searchKeyword[rule.key]"
                  type="text"
                  class="input text-sm"
                  :placeholder="t('admin.groups.uaBinding.searchPlaceholder')"
                  @input="searchAccounts(rule.key)"
                  @focus="onSearchFocus(rule.key)"
                />
                <div
                  v-if="openDropdownKey === rule.key"
                  class="absolute z-50 mt-1 max-h-48 w-full overflow-auto rounded-lg border bg-white shadow-lg dark:border-dark-600 dark:bg-dark-800"
                >
                  <p
                    v-if="!(searchResults[rule.key]?.length > 0)"
                    class="px-3 py-2 text-sm text-gray-400"
                  >
                    {{ t("admin.groups.uaBinding.searchEmpty") }}
                  </p>
                  <button
                    v-for="account in searchResults[rule.key] ?? []"
                    :key="account.id"
                    type="button"
                    class="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700"
                    :class="{ 'opacity-50': rule.accountIds.includes(account.id) }"
                    :disabled="rule.accountIds.includes(account.id)"
                    @click="selectAccount(rule, account)"
                  >
                    <span>{{ account.name }}</span>
                    <span class="ml-2 text-xs text-gray-400">#{{ account.id }}</span>
                  </button>
                </div>
              </div>
              <p class="mt-1 text-xs text-gray-400">
                {{ t("admin.groups.uaBinding.accountsHint") }}
              </p>
            </div>
          </div>
          <button
            type="button"
            class="mt-5 p-1.5 text-gray-400 transition-colors hover:text-red-500"
            :title="t('admin.groups.uaBinding.removeRule')"
            @click="removeRule(rule)"
          >
            <Icon name="trash" size="sm" />
          </button>
        </div>
      </div>

      <button
        type="button"
        class="flex items-center gap-1.5 text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
        data-testid="ua-binding-add-rule"
        @click="addRule"
      >
        <Icon name="plus" size="sm" />
        {{ t("admin.groups.uaBinding.addRule") }}
      </button>

      <p
        v-if="validationError"
        class="text-xs text-red-600 dark:text-red-400"
        role="alert"
      >
        {{ validationError }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import Icon from "@/components/icons/Icon.vue";
import Toggle from "@/components/common/Toggle.vue";
import { useKeyedDebouncedSearch } from "@/composables/useKeyedDebouncedSearch";
import { adminAPI } from "@/api/admin";
import type { GroupUABindingRule } from "@/types";

interface SimpleAccount {
  id: number;
  name: string;
}

interface RuleDraft {
  key: string;
  name: string;
  patternsText: string;
  accountIds: number[];
}

const props = defineProps<{
  groupId: number;
  /** 分组平台；composite 分组不按平台过滤账号 */
  platform: string;
}>();

const { t } = useI18n();

const enabled = ref(false);
const rules = ref<RuleDraft[]>([]);
const names = ref<Record<number, string>>({});
const validationError = ref("");

let keySeq = 0;
const nextKey = () => `ua-rule-${++keySeq}`;

const accountLabel = (id: number) => names.value[id] ?? `#${id}`;

const resolveNames = (ids: number[]) => {
  for (const id of ids) {
    if (names.value[id]) continue;
    adminAPI.accounts
      .getById(id)
      .then((account) => {
        names.value = { ...names.value, [id]: account.name };
      })
      .catch(() => {
        // 解析失败时展示 #<id>，提示管理员清理失效账号。
      });
  }
};

/** 用分组已保存的配置初始化（打开编辑对话框时调用）。 */
const load = (value: { enabled?: boolean; rules?: GroupUABindingRule[] | null }) => {
  enabled.value = value.enabled ?? false;
  rules.value = (value.rules ?? []).map((rule) => ({
    key: nextKey(),
    name: rule.name ?? "",
    patternsText: (rule.patterns ?? []).join("\n"),
    accountIds: [...(rule.account_ids ?? [])],
  }));
  names.value = {};
  validationError.value = "";
  searchKeyword.value = {};
  searchResults.value = {};
  openDropdownKey.value = "";
  resolveNames(rules.value.flatMap((rule) => rule.accountIds));
};

const parsePatterns = (text: string) =>
  text
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line !== "");

/** 当前配置（API 格式）；空规则（没有 UA 也没有账号）会被丢弃。 */
const getValue = (): { ua_binding_enabled: boolean; ua_binding_rules: GroupUABindingRule[] } => ({
  ua_binding_enabled: enabled.value,
  ua_binding_rules: rules.value
    .map((rule) => ({
      name: rule.name.trim(),
      patterns: parsePatterns(rule.patternsText),
      account_ids: [...rule.accountIds],
    }))
    .filter((rule) => rule.patterns.length > 0 || rule.account_ids.length > 0),
});

/** 校验：开启后至少一条规则，且每条规则都有 UA 和账号。 */
const validate = (): boolean => {
  validationError.value = "";
  if (!enabled.value) return true;
  const value = getValue().ua_binding_rules;
  if (value.length === 0) {
    validationError.value = t("admin.groups.uaBinding.atLeastOneRule");
    return false;
  }
  if (value.some((rule) => rule.patterns.length === 0 || rule.account_ids.length === 0)) {
    validationError.value = t("admin.groups.uaBinding.ruleIncomplete");
    return false;
  }
  return true;
};

const addRule = () => {
  rules.value.push({ key: nextKey(), name: "", patternsText: "", accountIds: [] });
};

const removeRule = (rule: RuleDraft) => {
  rules.value = rules.value.filter((item) => item !== rule);
};

// ---- 账号搜索（限定本分组内账号） ----
const containerRef = ref<HTMLElement | null>(null);
const searchKeyword = ref<Record<string, string>>({});
const searchResults = ref<Record<string, SimpleAccount[]>>({});
const openDropdownKey = ref("");

const handleDocumentClick = (event: MouseEvent) => {
  const target = event.target as HTMLElement;
  if (!target.closest?.(".relative") || !containerRef.value?.contains(target)) {
    openDropdownKey.value = "";
  }
};
onMounted(() => document.addEventListener("click", handleDocumentClick));
onUnmounted(() => document.removeEventListener("click", handleDocumentClick));

const searchRunner = useKeyedDebouncedSearch<SimpleAccount[]>({
  delay: 300,
  search: async (keyword, { signal }) => {
    const filters: { search: string; group: string; platform?: string } = {
      search: keyword,
      group: String(props.groupId),
    };
    if (props.platform && props.platform !== "composite") {
      filters.platform = props.platform;
    }
    const res = await adminAPI.accounts.list(1, 20, filters, { signal });
    return res.items.map((account) => ({ id: account.id, name: account.name }));
  },
  onSuccess: (key, result) => {
    searchResults.value[key] = result;
  },
  onError: (key) => {
    searchResults.value[key] = [];
  },
});

const searchAccounts = (key: string) => {
  openDropdownKey.value = key;
  searchRunner.trigger(key, searchKeyword.value[key] || "");
};

const onSearchFocus = (key: string) => {
  openDropdownKey.value = key;
  if (!searchResults.value[key]?.length) {
    searchAccounts(key);
  }
};

const selectAccount = (rule: RuleDraft, account: SimpleAccount) => {
  if (!rule.accountIds.includes(account.id)) {
    rule.accountIds.push(account.id);
    names.value = { ...names.value, [account.id]: account.name };
  }
  searchKeyword.value[rule.key] = "";
  openDropdownKey.value = "";
  validationError.value = "";
};

const removeAccount = (rule: RuleDraft, id: number) => {
  rule.accountIds = rule.accountIds.filter((accountId) => accountId !== id);
};

defineExpose({ load, getValue, validate });
</script>
