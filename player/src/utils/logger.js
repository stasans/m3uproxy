const isDevelopment = process.env.NODE_ENV === 'development';

const log = (message, level = 'info') => {
    if (isDevelopment) {
        console[level](message); // Log to the console in development
        if (/Philips|NETTV|SmartTvA|_TV_MT9288/i.test(navigator.userAgent)) {
            // Send logs to a server or external service in production
            const img = new Image();
            img.src = `http://${window.location.hostname}:3000/log?level=${level}&msg=${encodeURIComponent(message)}`;
        }
    }
};

export const Logger = {
    info: (msg) => log(msg, 'info'),
    warn: (msg) => log(msg, 'warn'),
    error: (msg) => log(msg, 'error'),
    debug: (msg) => log(msg, 'debug'),
};