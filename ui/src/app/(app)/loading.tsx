import { Loader2Icon } from "lucide-react";

export default function AppRootLoading() {
  return (
    <div className="container mt-96 flex flex-row justify-center">
      <Loader2Icon className="animate-spin h-20 w-20" />
    </div>
  );
}
