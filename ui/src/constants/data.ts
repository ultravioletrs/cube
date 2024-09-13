import type { NavItemGroup } from "@/types/navigation";

const domainNavbarGroup: NavItemGroup = {
  title: "Domain Management",
  navItems: [
    {
      title: "Domain",
      href: "/domain/info",
      label: "Dashboard",
      description: "Domain info",
      icon: "domain",
      disabled: false,
      external: false,
    },
    {
      title: "Members",
      href: "/domain/users",
      label: "Members",
      description: "Members",
      icon: "users",
      disabled: false,
      external: false,
    },
    {
      title: "Invitations",
      href: "/domain/invitations",
      label: "Invitations",
      description: "Invitations",
      icon: "invitations",
      disabled: false,
      external: false,
    },
  ],
};

export const sideNavBarGroups: NavItemGroup[] = [domainNavbarGroup];

export const sidebarExpandWidth = 14;
export const sidebarCollapseWidth = 4.5;
export const topbarHeight = 14;

export const enabledEntity = "#2a8c85";
export const disabledEntity = "#e76351";
