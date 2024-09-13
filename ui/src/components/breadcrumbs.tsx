import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import clsx from "clsx";
import React from "react";

export interface BreadcrumbProps {
  label: string;
  href: string;
  active?: boolean;
}

const Breadcrumbs = ({ breadcrumbs }: { breadcrumbs: BreadcrumbProps[] }) => {
  return (
    <Breadcrumb className="mb-4 text-2xl">
      <BreadcrumbList>
        {breadcrumbs.map((breadcrumb, index) => (
          <React.Fragment key={breadcrumb.label}>
            <BreadcrumbItem
              className={clsx(breadcrumb.active ? "font-bold" : "font-normal")}
            >
              <BreadcrumbLink href={breadcrumb.href}>
                {breadcrumb.label}
              </BreadcrumbLink>
            </BreadcrumbItem>
            {index !== breadcrumbs.length - 1 && <BreadcrumbSeparator />}
          </React.Fragment>
        ))}
      </BreadcrumbList>
    </Breadcrumb>
  );
};

export default Breadcrumbs;
