import { IsGoogleEnabled } from "@/lib/google";
import Image from "next/image";
import Link from "next/link";
import { GoogleProvider } from "./_components/google-provider";
import Loginform from "./_components/login-form";

const Login = async () => {
  const isGoogleEnabled = await IsGoogleEnabled();
  return (
    <div className="mt-20 pt-9 w-[430px] container bg-primary text-white rounded-lg pb-7">
      <>
        <h1 className="text-center block font-sans text-5xl font-semibold leading-tight tracking-normal text-inherit antialiased border-b-2">
          <Image
            src="abstract-machines_logo_landscape-white.svg"
            alt="Abstract Machines Logo"
            width="1000"
            height="80"
          />
        </h1>
        <h2 className="mt-5 text-center text-2xl font-bold leading-9 tracking-tight">
          Sign In
        </h2>
      </>
      <div className="mt-4">
        <Loginform />
      </div>
      {isGoogleEnabled && <GoogleProvider type="signin" />}
      <div className="text-sm mt-2 font-medium pb-12">
        Not registered?{" "}
        <Link href="/register" className="hover:underline font-bold">
          Sign up
        </Link>
      </div>
    </div>
  );
};

export default Login;
