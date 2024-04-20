/// <reference types="vite/client" />

import React from 'react';
import ReactDOM from 'react-dom/client';
import { OverlaysProvider } from '@blueprintjs/core';
import '@blueprintjs/core/lib/css/blueprint.css';
import './index.scss';
import App from './App.tsx';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <OverlaysProvider>
      <App />
    </OverlaysProvider>
  </React.StrictMode>,
);
