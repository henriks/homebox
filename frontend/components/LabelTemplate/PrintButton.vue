<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { toast } from "@/components/ui/sonner";
  import MdiPrinter from "~icons/mdi/printer";
  import MdiChevronDown from "~icons/mdi/chevron-down";
  import MdiLoading from "~icons/mdi/loading";
  import { Button, ButtonGroup } from "@/components/ui/button";
  import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
  } from "@/components/ui/dropdown-menu";
  import { useDialog } from "@/components/ui/dialog-provider";
  import { DialogID } from "@/components/ui/dialog-provider/utils";
  import { route } from "~/lib/api/base/urls";

  const props = withDefaults(
    defineProps<{
      /** What the label is for */
      type: "item" | "location";
      id: string;
      variant?: "default" | "outline";
    }>(),
    { variant: "default" }
  );

  const { t } = useI18n();
  const { openDialog } = useDialog();
  const { isQuickPrintAvailable, quickPrintItems, quickPrintLocations, defaultTemplateId, labelPrinting } =
    useQuickPrint();
  const { templates, pending: templatesPending, error: templatesError } = useLabelTemplates();
  const { render, renderLocations } = useLabelTemplateActions();
  const { selectedId } = useCollections();

  const busy = ref(false);

  const defaultTemplate = computed(() => templates.value?.find(tpl => tpl.id === defaultTemplateId.value) ?? null);
  const basicPrintIsDefault = computed(
    () => labelPrinting.value && !templatesPending.value && !templatesError.value && templates.value?.length === 0
  );

  function openChooser() {
    openDialog(DialogID.PrintLabelTemplate, {
      params: props.type === "item" ? { itemIds: [props.id] } : { locationIds: [props.id] },
    });
  }

  async function quickPrint() {
    if (basicPrintIsDefault.value) {
      await basicPrint();
      return;
    }
    if (!isQuickPrintAvailable.value) {
      openChooser();
      return;
    }
    busy.value = true;
    try {
      const ok = props.type === "item" ? await quickPrintItems([props.id]) : await quickPrintLocations([props.id]);
      if (ok) {
        toast.success(t("components.label_template.print_button.sent"));
      } else {
        openChooser();
      }
    } catch (e) {
      toast.error(t("components.label_template.toast.print_failed"), {
        description: (e as Error).message,
      });
    } finally {
      busy.value = false;
    }
  }

  async function basicPrint() {
    if (busy.value) return;
    busy.value = true;
    try {
      const params: Record<string, string | boolean> = { print: true };
      if (selectedId.value) params.tenant = selectedId.value;
      const kind = props.type === "location" ? "location" : "entity";
      const response = await fetch(route(`/labelmaker/${kind}/${props.id}`, params), { cache: "no-store" });
      if (!response.ok) throw new Error(await response.text());
      toast.success(t("components.label_template.print_button.sent"));
    } catch (e) {
      toast.error(t("components.label_template.toast.print_failed"), {
        description: (e as Error).message,
      });
    } finally {
      busy.value = false;
    }
  }

  async function downloadPng() {
    if (!defaultTemplateId.value) {
      openChooser();
      return;
    }
    busy.value = true;
    try {
      const blob =
        props.type === "item"
          ? await render(defaultTemplateId.value, [props.id])
          : await renderLocations(defaultTemplateId.value, [props.id]);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `label-${props.id}.png`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      toast.success(t("components.label_template.toast.downloaded"));
    } catch {
      toast.error(t("components.label_template.toast.render_failed"));
    } finally {
      busy.value = false;
    }
  }
</script>

<template>
  <div class="flex items-center">
    <ButtonGroup>
      <Button
        :variant="variant"
        class="w-9 md:w-auto"
        :disabled="busy"
        :aria-label="$t('components.label_template.print_button.print_label')"
        @click="quickPrint"
      >
        <MdiLoading v-if="busy" class="animate-spin" />
        <MdiPrinter v-else />
        <span class="hidden md:inline">{{ $t("components.label_template.print_button.print_label") }}</span>
      </Button>
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button
            :variant="variant"
            size="icon"
            class="w-8 border-l border-l-white/30"
            :disabled="busy"
            :aria-label="$t('components.label_template.print_button.more')"
          >
            <MdiChevronDown class="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" class="w-80">
          <template v-if="isQuickPrintAvailable && defaultTemplate">
            <DropdownMenuItem class="flex-col items-start gap-0.5" @click="quickPrint">
              <span class="font-medium">
                {{
                  $t("components.label_template.print_button.quick_with", {
                    template: defaultTemplate.name,
                  })
                }}
              </span>
              <span class="text-xs text-muted-foreground">
                {{ $t("components.label_template.print_button.quick_hint") }}
              </span>
            </DropdownMenuItem>
          </template>
          <DropdownMenuLabel v-else-if="!basicPrintIsDefault" class="font-normal text-muted-foreground">
            {{ $t("components.label_template.print_button.no_defaults") }}
          </DropdownMenuLabel>
          <DropdownMenuItem @click="openChooser">
            {{ $t("components.label_template.print_button.choose") }}
          </DropdownMenuItem>
          <DropdownMenuItem v-if="labelPrinting" :disabled="busy" @click="basicPrint">
            {{ $t("components.label_template.print_button.basic_label") }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="downloadPng">
            {{ $t("components.label_template.print_button.download_png") }}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem v-if="labelPrinting" as-child>
            <NuxtLink to="/label-templates" class="text-muted-foreground">
              {{
                $t(
                  defaultTemplate
                    ? "components.label_template.print_button.change_default"
                    : "components.label_template.print_button.set_default"
                )
              }}
            </NuxtLink>
          </DropdownMenuItem>
          <p v-else class="px-2 py-1.5 text-xs text-muted-foreground">
            {{ $t("components.label_template.print_button.server_print_hint") }}
          </p>
        </DropdownMenuContent>
      </DropdownMenu>
    </ButtonGroup>
  </div>
</template>
