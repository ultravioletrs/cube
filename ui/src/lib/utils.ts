import { type ClassValue, clsx } from "clsx";
import { DateTime } from "luxon";
import type { ReadonlyURLSearchParams } from "next/navigation";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export const createPageUrl = (
  searchParams: ReadonlyURLSearchParams,
  pathname: string,
  value: string | number,
  type: string,
) => {
  const params = new URLSearchParams(searchParams);
  params.set(type, value.toString());

  return `${pathname}?${params.toString()}`;
};

export const generateRandomColor = () => {
  const letters = "0123456789ABCDEF";
  let color = "#";
  for (let i = 0; i < 6; i++) {
    color += letters[Math.floor(Math.random() * 16)];
  }
  return color;
};

export const generateUniqueNumber = () => {
  const timeStamp = new Date().getTime();
  const randomNumber = Math.floor(Math.random() * 1000);
  return Number.parseInt(timeStamp.toString() + randomNumber.toString());
};

export function toSentenseCase(val: string) {
  return val[0].toUpperCase() + val.slice(1);
}

export const calculateInterval = (startTime: Date, endTime: Date): number => {
  return Math.abs((endTime.getTime() - startTime.getTime()) / 1000);
};

const hexToRgb = (hex: string): [number, number, number] => {
  let newHex = hex.replace(/^#/, "");
  if (newHex.length === 3) {
    newHex = newHex
      .split("")
      .map((hex) => hex + hex)
      .join("");
  }
  const num = Number.parseInt(newHex, 16);
  return [(num >> 16) & 255, (num >> 8) & 255, num & 255];
};

const rgbToHex = (r: number, g: number, b: number): string => {
  return `#${[r, g, b]
    .map((x) => {
      const hex = x.toString(16);
      return hex.length === 1 ? `0${hex}` : hex;
    })
    .join("")}`;
};

const adjustBrightness = (rgb: [number, number, number], factor: number) => {
  return rgb.map((component) => {
    const value = Math.min(255, Math.max(0, component + factor));
    return value;
  }) as [number, number, number];
};

export const adjustColorForTheme = (color: string, theme: string): string => {
  if (theme !== "dark") {
    return color;
  }
  const rgb = hexToRgb(color);
  const threshold = 100;
  const isDark = rgb.reduce((acc, val) => acc + val, 0) / 3 < threshold;

  if (isDark) {
    const brightenedRgb = adjustBrightness(rgb, 150);
    return rgbToHex(...brightenedRgb);
  }

  return color;
};

export const timeFormatter = (unixTime: number, timeFormat: string) => {
  const myDateTime = DateTime.fromMillis(unixTime / 1000000);
  return myDateTime.toFormat(timeFormat) || "";
};

export const darkDialogTheme = {
  base00: "#020817", // Background color
  base01: "#282a36", // Slightly lighter background for sections
  base02: "#44475a", // Even lighter for nested sections
  base03: "#6272a4", // Comments, invisibles, line highlighting
  base04: "#f8f8f2", // Default text color
  base05: "#f8f8f2", // Default text color
  base06: "#bd93f9", // Variables, XML Tags, Markup Link Text
  base07: "#50fa7b", // String, RegExp, Escape, Tag
  base08: "#ff79c6", // Number, boolean, constant, inline code
  base09: "#8be9fd", // Attribute, class, function name
  base0A: "#f1fa8c", // Keywords, storage, selector, markup italic, diff added
  base0B: "#ffb86c", // Class name, built-in constant, mark
  base0C: "#ff5555", // Function argument, tag attribute, diff change
  base0D: "#8be9fd", // Highlighted
  base0E: "#bd93f9", // Regex, important, markup bold
  base0F: "#ffb86c", // Deprecated, opening/closing embedded tags
};

export const lightDialogTheme = {
  base00: "#ffffff", // Default background color
  base01: "#f5f5f5",
  base02: "#e0e0e0",
  base03: "#d6d6d6",
  base04: "#4d4d4c",
  base05: "#5e5e5e",
  base06: "#d6d6d6",
  base07: "#1d1f21",
  base08: "#c82829",
  base09: "#f5871f",
  base0A: "#eab700",
  base0B: "#718c00",
  base0C: "#3e999f",
  base0D: "#4271ae",
  base0E: "#8959a8",
  base0F: "#a3685a",
};
