"use client";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { GenerateGoogleUrl } from "@/lib/google";
import { useRouter } from "next/navigation";
export const GoogleProvider = ({ type }: { type: string }) => {
  const router = useRouter();
  let value = "";
  switch (type) {
    case "signin": {
      value = "Sign in";
      break;
    }
    case "signup": {
      value = "Sign up";
      break;
    }
  }
  return (
    <div className="flex flex-col gap-2 mt-4">
      <div className="flex flex-row w-full items-center gap-2 justify-center">
        <Separator className="w-1/3" />
        <span className="text-center">or</span>
        <Separator className="w-1/3" />
      </div>
      <Button
        variant="secondary"
        onClick={async () => {
          const url = await GenerateGoogleUrl();
          router.push(url);
        }}
      >
        {`${value} with Google`}
      </Button>
      <Separator className="mt-4" />
    </div>
  );
};
