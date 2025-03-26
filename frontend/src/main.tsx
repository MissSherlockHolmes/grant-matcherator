import { createRoot } from 'react-dom/client'
import App from './App.tsx'
import './index.css'

console.log('=== App Starting ===');
console.log('Root element:', document.getElementById("root"));

const root = createRoot(document.getElementById("root")!);
console.log('Root created');

root.render(<App />);
console.log('App rendered');
