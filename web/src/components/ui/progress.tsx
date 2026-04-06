import * as React from "react"

import { cn } from "#/lib/utils"

function Progress({
  className,
  value,
  max = 100,
  indicatorClassName,
  ...props
}: React.ComponentProps<"div"> & {
  value?: number
  max?: number
  indicatorClassName?: string
}) {
  const percentage = value != null ? Math.min(Math.max((value / max) * 100, 0), 100) : 0

  return (
    <div
      data-slot="progress"
      role="progressbar"
      aria-valuenow={value}
      aria-valuemin={0}
      aria-valuemax={max}
      className={cn(
        "bg-primary/20 relative h-2 w-full overflow-hidden rounded-full",
        className
      )}
      {...props}
    >
      <div
        data-slot="progress-indicator"
        className={cn(
          "h-full w-full flex-1 transition-all duration-300 ease-in-out",
          indicatorClassName || "bg-primary"
        )}
        style={{ transform: `translateX(-${100 - percentage}%)` }}
      />
    </div>
  )
}

export { Progress }