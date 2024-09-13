import { HttpError } from "@/types/errors";
import { z } from "zod";

export function getErrorMessage(err: unknown) {
  const unknownError = "Something went wrong, please try again later.";

  if (err instanceof z.ZodError) {
    const errors = err.issues.map((issue) => {
      return issue.message;
    });
    return errors.join("\n");
  }
  if (err instanceof HttpError) {
    return err.error || err.message || err.toString();
  }
  if (err instanceof Error) {
    return err.message;
  }
  if (err && typeof err === "object" && "message" in err) {
    return err.message;
  }
  if (err && typeof err === "object" && "error" in err) {
    return err.error;
  }
  if (typeof err === "string") {
    return err;
  }
  return unknownError;
}
