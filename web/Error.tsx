import ReactDOM from 'react-dom/client';
import { useState } from 'react';
import { Alert } from '@blueprintjs/core';

export function Error({
  error,
  root,
  container,
}: {
  error: string;
  root: ReactDOM.Root;
  container: HTMLDivElement;
}) {
  const [open, setOpen] = useState(true);
  return (
    <Alert
      isOpen={open}
      canEscapeKeyCancel
      onClose={() => {
        setOpen(false);
        // https://blueprintjs.com/docs/#core/components/alert
        setTimeout(() => {
          root.unmount();
          container.remove();
        }, 300);
      }}
    >
      <p>{error}</p>
    </Alert>
  );
}
