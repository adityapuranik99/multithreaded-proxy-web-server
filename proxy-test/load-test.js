const axios = require('axios');
const fs = require('fs').promises;

// Test Configuration
const CONFIG = {
    stages: [
        { duration: 60, requests: 1000 },    // Warmup: 1000 concurrent for 1 minute
        { duration: 120, requests: 2500 },   // Stage 1: Ramp up to 2500 concurrent for 2 minutes
        { duration: 180, requests: 5000 },   // Stage 2: Heavy load with 5000 concurrent for 3 minutes
        { duration: 60, requests: 7500 },    // Stage 3: Peak with 7500 concurrent for 1 minute
        { duration: 120, requests: 3000 },   // Stage 4: Scale down to 3000 concurrent for 2 minutes
        { duration: 60, requests: 1000 }     // Cool down: Back to 1000 concurrent for 1 minute
    ],
    proxyHost: 'localhost',
    proxyPort: 8080,
    timeoutMs: 10000,
    statsInterval: 5000,  // Log stats every 5 seconds
};

const TEST_URLS = {
    simple: [
        'http://example.com',
        'https://example.com'
    ],
    static: [
        'http://neverssl.com',
        'https://httpbin.org/status/200'
    ],
    complex: [
        'http://httpbin.org/get',
        'https://httpbin.org/get'
    ]
};

// Enhanced statistics tracking
const stats = {
    successful: 0,
    failed: 0,
    startTime: Date.now(),
    lastReportTime: Date.now(),
    timePoints: [],  // For graphing
    currentConcurrent: 0,
    responseTimes: [],
    errors: {},
    urlStats: {},
    throughputHistory: []
};

function createRequestConfig(url) {
    return {
        url,
        proxy: {
            host: CONFIG.proxyHost,
            port: CONFIG.proxyPort,
            protocol: 'http'
        },
        headers: {
            'User-Agent': 'load-test/1.0',
            'Accept': '*/*',
            'Connection': 'close'
        },
        maxRedirects: 0,
        validateStatus: null,
        timeout: CONFIG.timeoutMs,
        decompress: true
    };
}

async function makeRequest(url) {
    stats.currentConcurrent++;
    const startTime = Date.now();

    if (!stats.urlStats[url]) {
        stats.urlStats[url] = {
            attempts: 0,
            success: 0,
            failed: 0,
            totalTime: 0,
            responseTimeBuckets: new Array(10).fill(0)  // For response time distribution
        };
    }

    stats.urlStats[url].attempts++;

    try {
        const response = await axios(createRequestConfig(url));
        const responseTime = Date.now() - startTime;
        
        // Update statistics
        stats.successful++;
        stats.urlStats[url].success++;
        stats.urlStats[url].totalTime += responseTime;
        stats.responseTimes.push(responseTime);

        // Track response time distribution
        const bucket = Math.min(Math.floor(responseTime / 100), 9);
        stats.urlStats[url].responseTimeBuckets[bucket]++;

        return { success: true, time: responseTime, status: response.status };
    } catch (error) {
        const errorTime = Date.now() - startTime;
        stats.failed++;
        stats.urlStats[url].failed++;
        const errorMessage = error.code || error.message;
        stats.errors[errorMessage] = (stats.errors[errorMessage] || 0) + 1;
        return { success: false, error: errorMessage, time: errorTime };
    } finally {
        stats.currentConcurrent--;
    }
}

async function recordMetrics() {
    const currentTime = Date.now();
    const timePoint = {
        timestamp: currentTime,
        concurrent: stats.currentConcurrent,
        successful: stats.successful,
        failed: stats.failed,
        avgResponseTime: stats.responseTimes.length > 0 
            ? stats.responseTimes.reduce((a, b) => a + b) / stats.responseTimes.length 
            : 0
    };
    
    stats.timePoints.push(timePoint);
    stats.responseTimes = [];  // Reset for next interval

    // Calculate current throughput
    const throughput = (stats.successful + stats.failed) / 
                      ((currentTime - stats.lastReportTime) / 1000);
    stats.throughputHistory.push({
        timestamp: currentTime,
        throughput: throughput
    });
    
    stats.lastReportTime = currentTime;

    // Log current metrics
    console.log(`[${new Date().toISOString()}] ` +
        `Concurrent: ${stats.currentConcurrent}, ` +
        `Throughput: ${throughput.toFixed(2)} req/s, ` +
        `Success Rate: ${((stats.successful / (stats.successful + stats.failed)) * 100).toFixed(1)}%, ` +
        `Avg Response: ${timePoint.avgResponseTime.toFixed(2)}ms`);
}

async function saveFinalReport() {
    const report = {
        config: CONFIG,
        finalStats: {
            duration: (Date.now() - stats.startTime) / 1000,
            totalRequests: stats.successful + stats.failed,
            successRate: (stats.successful / (stats.successful + stats.failed)) * 100,
            errors: stats.errors,
            urlStats: stats.urlStats
        },
        timeSeriesData: stats.timePoints,
        throughputHistory: stats.throughputHistory
    };

    await fs.writeFile('load-test-report.json', JSON.stringify(report, null, 2));
    console.log('Detailed report saved to load-test-report.json');
}

async function runLoadTest() {
    console.log(`Starting heavy load test...
Configuration:
- Test Duration: ${CONFIG.stages.reduce((acc, stage) => acc + stage.duration, 0)} seconds
- Max Concurrent: ${Math.max(...CONFIG.stages.map(s => s.requests))}
- Proxy: http://${CONFIG.proxyHost}:${CONFIG.proxyPort}
`);

    // Start metrics recording
    const metricsInterval = setInterval(recordMetrics, CONFIG.statsInterval);

    // Run through each stage
    for (const [index, stage] of CONFIG.stages.entries()) {
        console.log(`\nStarting Stage ${index + 1}: ${stage.requests} concurrent requests for ${stage.duration} seconds`);
        
        const stageEnd = Date.now() + (stage.duration * 1000);
        
        while (Date.now() < stageEnd) {
            // Maintain target concurrent requests
            while (stats.currentConcurrent < stage.requests) {
                const urlType = ['simple', 'static', 'complex'][Math.floor(Math.random() * 3)];
                const urls = TEST_URLS[urlType];
                const url = urls[Math.floor(Math.random() * urls.length)];
                
                makeRequest(url).catch(console.error);  // Fire and forget
            }
            await new Promise(resolve => setTimeout(resolve, 100));  // Small delay to prevent CPU overload
        }
    }

    clearInterval(metricsInterval);
    await saveFinalReport();
}

console.log('Heavy Load Test Starting...');
runLoadTest()
    .then(() => console.log('Test completed'))
    .catch(console.error)
    .finally(() => process.exit(0));