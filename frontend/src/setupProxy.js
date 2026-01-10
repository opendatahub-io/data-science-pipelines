const { createProxyMiddleware } = require('http-proxy-middleware');

// Create React App will automatically use this setupProxy.js file,
// and you can configure the backend port via environment variable
module.exports = function(app) {
    // Read from environment variable with default fallback
    const backendPort = process.env.REACT_APP_BACKEND_PORT || '3001';
    const target = `http://localhost:${backendPort}`;

    console.log(`🔗 Proxying API requests to: ${target}`);

    app.use(
        '/apis',
        createProxyMiddleware({
            target,
            changeOrigin: true,
        })
    );

    app.use(
        '/system/*',
        createProxyMiddleware({
            target,
            changeOrigin: true,
        })
    );

    app.use(
        '/artifacts/*',
        createProxyMiddleware({
            target,
            changeOrigin: true,
        })
    );

    app.use(
        '/k8s/*',
        createProxyMiddleware({
            target,
            changeOrigin: true,
        })
    );
};