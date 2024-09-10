import type { ErrorBoundaryProps } from "hwy";
import { Paragraph } from "./paragraph.js";

function FallbackErrorBoundary(props: ErrorBoundaryProps) {
  return (
    <div class="flex-col-wrapper">
      <Paragraph>Whoops, something went wrong. Sorry about that.</Paragraph>
      <Paragraph>
        If you're feeling generous, please file an issue telling us what
        happened.
      </Paragraph>
    </div>
  );
}

export { FallbackErrorBoundary };
