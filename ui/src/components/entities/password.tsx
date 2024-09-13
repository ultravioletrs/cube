import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { EyeIcon, EyeOffIcon } from "lucide-react";
import { useState } from "react";
import type { ControllerRenderProps } from "react-hook-form";

type RegisterController = ControllerRenderProps<
  {
    email: string;
    password: string;
    name: string;
  },
  "password"
>;

type LoginController = ControllerRenderProps<
  {
    email: string;
    password: string;
  },
  "password"
>;

type ResetPasswordController = ControllerRenderProps<
  {
    password: string;
    confirmPassword: string;
  },
  "confirmPassword"
>;

type UpdatePasswordController = ControllerRenderProps<{
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
}>;

type Props = {
  processing: boolean;
  field:
    | RegisterController
    | LoginController
    | ResetPasswordController
    | UpdatePasswordController;
};

export function PasswordFormInput({ processing, field }: Props) {
  const [seePassword, toggleSeePassword] = useState(false);

  return (
    <div className="flex items-center space-x-2">
      <Input
        disabled={processing}
        type={seePassword ? "text" : "password"}
        placeholder="Enter password"
        {...field}
        className="text-card-foreground"
      />
      <Button
        type="button"
        variant="outline"
        size="icon"
        onClick={() => toggleSeePassword((prev) => !prev)}
      >
        {seePassword ? (
          <EyeOffIcon className="h-4 w-4 text-zinc-700" />
        ) : (
          <EyeIcon className="h-4 w-4 text-zinc-700" />
        )}
      </Button>
    </div>
  );
}
