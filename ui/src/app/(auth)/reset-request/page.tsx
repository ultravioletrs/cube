import ResetPasswordForm from "./_components/reset-password-form";

type ParamsProps = {
  searchParams: { token: string };
};

const ResetPasswordPage = ({ searchParams }: ParamsProps) => {
  const token = searchParams?.token as string;

  return (
    <>
      <div className="sm:mx-auto sm:w-full sm:max-w-sm">
        <h1 className="text-center block font-sans text-5xl font-semibold leading-tight tracking-normal text-inherit antialiased border-b-2 pb-3">
          Magistrala
        </h1>
        <h2 className="mt-5 text-center text-2xl font-bold leading-9 tracking-tight">
          Reset Password
        </h2>
      </div>
      {token ? (
        <div className="mt-10 sm:mx-auto sm:w-full sm:max-w-sm">
          <ResetPasswordForm searchParams={{ token }} />
        </div>
      ) : (
        <div className="sm:mx-auto sm:w-full sm:max-w-sm">
          <p className="mt-5 text-center text-lg font-semibold leading-9 tracking-tight">
            Missing token in the reset password link
          </p>
        </div>
      )}
    </>
  );
};

export default ResetPasswordPage;
