const axios = require('axios');

const testUrl = 'http://example.com';
const proxyUrl = 'http://localhost:8080';

async function testProxy() {
    try {
        console.log(`Testing proxy ${proxyUrl} with request to ${testUrl}`);
        const start = Date.now();
        
        // Configure axios more like curl
        const response = await axios.get(testUrl, {
            proxy: {
                host: 'localhost',
                port: 8080,
                protocol: 'http'
            },
            headers: {
                'User-Agent': 'curl/7.79.1',
                'Accept': '*/*'
            },
            maxRedirects: 0,  // Don't follow redirects automatically
            validateStatus: null,  // Don't throw on any status code
            timeout: 5000  // 5 second timeout
        });
        
        const time = Date.now() - start;
        console.log('Response received:');
        console.log(`Status: ${response.status}`);
        console.log(`Time: ${time}ms`);
        console.log('Headers:', response.headers);
        
    } catch (error) {
        console.error('Error details:');
        console.error('Message:', error.message);
        console.error('Code:', error.code);
        
        if (error.response) {
            console.error('Response status:', error.response.status);
            console.error('Response headers:', error.response.headers);
        }
        
        // Log axios configuration
        if (error.config) {
            console.error('Request configuration:', {
                url: error.config.url,
                method: error.config.method,
                headers: error.config.headers,
                proxy: error.config.proxy
            });
        }
    }
}

testProxy();