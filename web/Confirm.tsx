import ReactDOM from 'react-dom/client';
import { ReactNode, useState } from 'react';
import { Button, Dialog, DialogBody, DialogFooter, Intent } from '@blueprintjs/core';

export function Confirm({
  title,
  children,
  intent,
  callback,
  root,
  container,
}: {
  title: string;
  children: ReactNode;
  intent: Intent;
  callback: () => Promise<void>;
  root: ReactDOM.Root;
  container: HTMLDivElement;
}) {
  const [open, setOpen] = useState(true);
  const [loading, setLoading] = useState(false);
  const onClose = () => {
    setOpen(false);
    // https://blueprintjs.com/docs/#core/components/alert
    setTimeout(() => {
      root.unmount();
      container.remove();
    }, 300);
  };
  const onConfirm = async () => {
    setLoading(true);
    try {
      await callback();
      onClose();
    } finally {
      setLoading(false);
    }
  };
  return (
    <Dialog
      title={title}
      isOpen={open}
      isCloseButtonShown={false}
      onClose={onClose}
      canEscapeKeyClose
      canOutsideClickClose
    >
      <DialogBody>
        <div
          ref={body => body?.getElementsByTagName('input')[0]?.focus()}
          onKeyDown={evt => evt.key === 'Enter' && onConfirm()}
        >
          {children}
        </div>
      </DialogBody>
      <DialogFooter
        actions={
          <>
            <Button className="select-none" text="Cancel" onClick={onClose} />
            <Button
              className="select-none"
              intent={intent}
              loading={loading}
              text="OK"
              onClick={onConfirm}
            />
          </>
        }
      />
    </Dialog>
  );
}
