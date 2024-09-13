"use client";
import { CheckCheck, CopyIcon } from "lucide-react";
import { useState } from "react";
import { Button } from "./ui/button";

export const CopyButton = ({ data }: { data: string }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(data);
      setCopied(true);
      setTimeout(() => {
        setCopied(false);
      }, 2000);
    } catch (error) {
      console.error("Failed to copy", error);
    }
  };

  return (
    <Button type="button" size="sm" variant="ghost" onClick={handleCopy}>
      <span className="sr-only">Copy</span>
      {copied ? (
        <CheckCheck className="h-4 w-4" color="green" />
      ) : (
        <CopyIcon className="h-4 w-4" />
      )}
    </Button>
  );
};
