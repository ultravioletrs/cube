"use client";

import { Button } from "@/components/ui/button";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ForgotPasswordForm } from "./_components/forgot-password-form";

const ResetPasswordPage = () => {
  const router = useRouter();
  return (
    <>
      <div className="sm:mx-auto sm:w-full sm:max-w-sm">
        <h1 className="text-center block font-sans text-5xl font-semibold leading-tight tracking-normal text-inherit antialiased border-b-2 pb-3">
          Magistrala
        </h1>
        <h2 className="mt-5 text-center text-2xl font-bold leading-9 tracking-tight">
          Forgot Password?
        </h2>
      </div>
      <div className="mt-10 sm:mx-auto sm:w-full sm:max-w-sm">
        <ForgotPasswordForm />
      </div>
      <div className="text-center text-sm mt-5 font-medium">
        <Link href="/register" className="hover:underline">
          Create new account{" "}
        </Link>
      </div>
      <div className="mt-6">
        <Button
          variant="link"
          className="w-full text-center text-muted-foreground"
          onClick={() => router.push("/login")}
        >
          Back to login
        </Button>
      </div>
    </>
  );
};

export default ResetPasswordPage;
