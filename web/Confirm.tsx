import ReactDOM from 'react-dom/client';
import { useState } from 'react';
import { Alert } from '@blueprintjs/core';

export function Confirm({
  text,
  callback,
  root,
  container,
}: {
  text: string;
  callback: () => void;
  root: ReactDOM.Root;
  container: HTMLDivElement;
}) {
  const [open, setOpen] = useState(true);
  return (
    <Alert
      isOpen={open}
      cancelButtonText="Cancel"
      confirmButtonText="Yes"
      canEscapeKeyCancel
      canOutsideClickCancel
      onClose={confirmed => {
        if (confirmed) callback();
        setOpen(false);
        // https://blueprintjs.com/docs/#core/components/alert
        setTimeout(() => {
          root.unmount();
          container.remove();
        }, 300);
      }}
    >
      <p>{text}</p>
    </Alert>
  );
}
