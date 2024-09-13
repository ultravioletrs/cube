"use client";

import { Button, buttonVariants } from "@/components/ui/button";
import { CircleAlert } from "lucide-react";
import Link from "next/link";

const ErrorComponent = ({
  link,
  linkText,
  error,
  showLinkButton = true,
}: {
  link?: string;
  linkText?: string;
  error?: Error | string;
  showLinkButton?: boolean;
}) => {
  const handleReload = () => {
    window.location.reload();
  };

  if (typeof error === "string") {
    error = new Error(error);
  }

  return (
    <div className="flex flex-col items-center justify-center h-full w-full">
      <CircleAlert className="h-12 w-12 text-red-500" />
      <p className="font-medium text-gray-700 dark:text-gray-300 text-[30px] mb-4 mt-4">
        Oops!
      </p>
      <p className="font-medium text-gray-700 dark:text-gray-300 text-[20px]">
        Something went wrong
      </p>
      {error && (
        <p className=" text-red-600 text-sm mt-2">Error: {error.message}</p>
      )}
      <Button
        variant="link"
        onClick={handleReload}
        className="text-popover-foreground"
      >
        Please try again
      </Button>
      {showLinkButton && (
        <>
          <p className="mt-2">OR</p>
          <Link
            href={link as string}
            className={buttonVariants({
              variant: "default",
              className: "mt-4",
            })}
          >
            {linkText}
          </Link>
        </>
      )}
    </div>
  );
};

export default ErrorComponent;
