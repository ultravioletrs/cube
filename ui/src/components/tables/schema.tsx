import { Icons } from "@/components/icons";
import { disabledEntity, enabledEntity } from "@/constants/data";
export const statusSchema = [
  {
    value: "enabled",
    label: "Enabled",
    icon: Icons.enabled,
    colour: enabledEntity,
  },
  {
    value: "disabled",
    label: "Disabled",
    icon: Icons.disabled,
    colour: disabledEntity,
  },
  {
    value: "all",
    label: "All",
    icon: Icons.all,
    colour: "grey",
  },
];
