import React from 'react';
import ReactDOM from 'react-dom/client';
import { FormBuilderApp } from './App';

// Render the form builder app
const root = ReactDOM.createRoot(document.getElementById('root') as HTMLElement);
root.render(
  <React.StrictMode>
    <FormBuilderApp />
  </React.StrictMode>
);
