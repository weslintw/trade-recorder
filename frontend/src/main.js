// Enable global console timestamps
(function () {
  const originalLog = console.log;
  const originalWarn = console.warn;
  const originalError = console.error;

  const getTimestamp = () => {
    const now = new Date();
    return `[${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}.${now.getMilliseconds().toString().padStart(3, '0')}]`;
  };

  console.log = function (...args) {
    originalLog.apply(console, [getTimestamp(), ...args]);
  };
  console.warn = function (...args) {
    originalWarn.apply(console, [getTimestamp(), ...args]);
  };
  console.error = function (...args) {
    originalError.apply(console, [getTimestamp(), ...args]);
  };
})();

import App from './App.svelte'

const app = new App({
  target: document.getElementById('app'),
})

export default app

