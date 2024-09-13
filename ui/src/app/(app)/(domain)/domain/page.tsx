import ErrorComponent from "@/components/error-component";
import { GetHomePageData } from "@/lib/homepage";

export default async function Domain() {
  const response = await GetHomePageData();

  return (
    <>
      {response.error !== null ? (
        <div className="mt-40">
          <ErrorComponent
            link="/"
            linkText="Go Back to Domain Login"
            error={response.error}
          />
        </div>
      ) : (
        <div className="mt-40">
          <p>Entity not found</p>
        </div>
      )}
    </>
  );
}
