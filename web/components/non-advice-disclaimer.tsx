import { cn } from "@/lib/cn";

const DEFAULT_TEXT =
  "Isto é conteúdo educacional, não recomendação de investimento. As decisões são suas.";
const EXTENDED_TEXT =
  "As projeções são estimativas não garantidas, baseadas em premissas explícitas e nos dados atuais da sua carteira. YieldForge não emite ordens de compra ou venda, metas de preço ou quantidades. Consulte um profissional habilitado antes de decidir.";

/**
 * Required footer on any surface that renders AI output (FR-014, non-advice). Its
 * presence is a contract of the view, not an optional prop.
 */
export function NonAdviceDisclaimer({
  extended = false,
  className,
}: {
  extended?: boolean;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex items-start gap-2 rounded-md border border-hairline bg-elevated px-3.5 py-2.5 text-xs font-medium leading-relaxed text-muted-strong",
        className,
      )}
    >
      <span
        aria-hidden
        className="mt-px flex h-[15px] w-[15px] flex-none items-center justify-center rounded-full border border-info text-[10px] font-bold text-info"
      >
        i
      </span>
      <span>{extended ? EXTENDED_TEXT : DEFAULT_TEXT}</span>
    </div>
  );
}
