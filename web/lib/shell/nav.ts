import type { ComponentType, SVGProps } from "react";
import {
  IconPainel,
  IconCarteira,
  IconInsights,
  IconSaude,
  IconProjecoes,
  IconPerfil,
} from "@/components/shell/icons";

export interface NavItem {
  href: string;
  label: string;
  Icon: ComponentType<SVGProps<SVGSVGElement>>;
}

// The authenticated nav. Chat/Copiloto is deliberately absent — it is a global floating
// widget, not a route (SPEC-215). (SPEC-200 FR-2006)
export const NAV_ITEMS: NavItem[] = [
  { href: "/dashboard", label: "Painel", Icon: IconPainel },
  { href: "/portfolio", label: "Carteira", Icon: IconCarteira },
  { href: "/insights", label: "Insights", Icon: IconInsights },
  { href: "/health", label: "Saúde", Icon: IconSaude },
  { href: "/projections", label: "Projeções", Icon: IconProjecoes },
  { href: "/profile", label: "Perfil", Icon: IconPerfil },
];

export function isActive(pathname: string, href: string): boolean {
  return pathname === href || pathname.startsWith(href + "/");
}
