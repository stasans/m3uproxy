import './main.css';
import React from 'react';
import App from './App';
import { GlobalConfigProvider } from './GlobalConfigContext';
import { createRoot } from 'react-dom/client';


const container = document.getElementById('app');
const root = createRoot(container); // createRoot(container!) if you use TypeScript
root.render(
    <GlobalConfigProvider>
        <App tab="home" />
    </GlobalConfigProvider>,
);