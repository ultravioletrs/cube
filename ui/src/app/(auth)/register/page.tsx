import { IsGoogleEnabled } from "@/lib/google";
import Image from "next/image";
import Link from "next/link";
import { GoogleProvider } from "../login/_components/google-provider";
import RegisterForm from "./_components/register-form";

const Register = async () => {
  const isGoogleEnabled = await IsGoogleEnabled();
  return (
    <div className="mt-20 pt-9 w-[430px] container bg-primary tealtide:bg-primary text-white rounded-lg pb-14">
      <>
        <div className="sm:mx-auto sm:w-full sm:max-w-sm">
          <Image
            src="abstract-machines_logo_landscape-white.svg"
            alt="Abstract Machines Logo"
            width="1000"
            height="80"
            className="border-b-2"
          />
          <h2 className="text-center text-2xl font-bold leading-9 tracking-tight mt-5">
            Sign Up
          </h2>
        </div>
      </>
      <div className="mt-3">
        <RegisterForm />
      </div>
      {isGoogleEnabled && <GoogleProvider type="signup" />}
      <div className="text-sm pt-1 font-medium text-center">
        Already have an account?{" "}
        <Link href="/login" className=" hover:underline">
          Sign in
        </Link>
      </div>
    </div>
  );
};

export default Register;
