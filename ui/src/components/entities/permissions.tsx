import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

export function ViewPermissions({ permissions }: { permissions: string[] }) {
  const badgeVariant = (permission: string): string => {
    switch (permission) {
      case "admin":
        return "bg-red-700 text-white";
      case "edit":
        return "bg-fuchsia-700 text-white";
      case "membership":
        return "bg-teal-700 text-white";
      case "share":
        return "bg-emerald-500 text-white";
      default:
        return "";
    }
  };
  return (
    <div className="flex space-x-2">
      <span className="max-w-[900px] font-medium">
        {permissions?.map((permission) => (
          <Badge
            key={permission}
            variant="outline"
            className={cn(
              "rounded-sm px-1 font-normal text-xs mx-1 my-1",
              badgeVariant(permission),
            )}
          >
            {permission}
          </Badge>
        ))}
      </span>
    </div>
  );
}
