"use client";

import { Icons } from "@/components/icons";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { UpdateMetadata } from "@/lib/entities";
import { EntityType, type Metadata } from "@/types/entities";
import { Themes } from "@/types/entities";
import type { User } from "@absmach/magistrala-sdk";
import { useTheme } from "next-themes";
import { useEffect } from "react";

export const ThemeToggle = ({ user }: { user: User }) => {
  const { theme, setTheme, resolvedTheme } = useTheme();
  const metadata = user?.metadata as Metadata;
  const id = user?.id as string;
  if (metadata) {
    if (!metadata.ui) {
      metadata.ui = {};
    }
  }

  if (!metadata?.ui?.theme) {
    setTheme(Themes.Default);
  }

  const handleThemeChange = async (newTheme: string) => {
    const themeToSet = newTheme || Themes.Default;
    setTheme(themeToSet);
    const updatedMetadata = {
      ...metadata,
      ui: {
        ...metadata?.ui,
        theme: themeToSet,
      },
    };
    await UpdateMetadata(id, EntityType.User, updatedMetadata);
  };

  useEffect(() => {
    if (metadata?.ui?.theme && metadata.ui?.theme !== resolvedTheme) {
      setTheme(metadata?.ui?.theme);
    }
  }, [metadata?.ui?.theme, resolvedTheme, setTheme]);

  const renderEntityIcon = () => {
    switch (theme) {
      case Themes.Default:
        return <Icons.sun className="size-5" />;
      case Themes.MidnightSky:
        return <Icons.moon className="size-5" />;
      case Themes.TealTide:
        return <Icons.shell className="size-5" />;
      default:
        return <Icons.sun className="size-5" />;
    }
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild={true}>
        <Button
          variant="outline"
          size="icon"
          className="hover:bg-primary/10 dark:hover:bg-accent"
        >
          {renderEntityIcon()}
          <span className="sr-only">Toggle theme</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem
          onClick={() => handleThemeChange(Themes.Default)}
          className="hover:bg-primary/10 dark:hover:bg-accent focus:bg-primary/10"
        >
          Default
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => handleThemeChange(Themes.MidnightSky)}
          className="hover:bg-primary/10 dark:hover:bg-accent focus:bg-primary/10"
        >
          Midnight Sky
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => handleThemeChange(Themes.TealTide)}
          className="hover:bg-primary/10 dark:hover:bg-accent focus:bg-primary/10"
        >
          Teal Tide
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
