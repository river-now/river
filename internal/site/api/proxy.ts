import type { VercelRequest, VercelResponse } from "@vercel/node";
import { ChildProcess, spawn } from "node:child_process";
import http from "node:http";
import { join } from "node:path";
import waveConfig from "../app/wave.config.json" with { type: "json" };

logInfo("Initializing proxy...");

const GO_APP_LOCATION = join(
	process.cwd(),
	"internal/site",
	waveConfig.Core.DistDir,
	"main",
);
const GO_APP_HEALTH_CHECK_ENDPOINT = waveConfig.Watch.HealthcheckEndpoint;
const GO_APP_STARTUP_TIMEOUT_IN_MS = 10_000; // 10s
const PORT = 8080;

let goProcess: ChildProcess | null = null;
let goReady = false;

async function waitForGoApp(url: string, timeout: number): Promise<void> {
	const startTime = Date.now();
	return new Promise((resolve, reject) => {
		const attempt = () => {
			http.get(url, (res) => {
				if (
					res.statusCode &&
					res.statusCode >= 200 &&
					res.statusCode < 400
				) {
					resolve();
				} else {
					scheduleNextAttempt();
				}
			}).on("error", scheduleNextAttempt);
		};

		const scheduleNextAttempt = (err?: Error) => {
			if (Date.now() - startTime > timeout) {
				return reject(err || new Error("Health check timed out."));
			}
			setTimeout(attempt, 50);
		};

		attempt();
	});
}

async function init() {
	const startTime = performance.now();

	try {
		// Add extensive debugging here
		logInfo("Current working directory:", process.cwd());
		logInfo("GO_APP_LOCATION:", GO_APP_LOCATION);
		logInfo("waveConfig.Core.DistDir:", waveConfig.Core.DistDir);

		// Check if the file exists and is accessible
		const fs = await import("node:fs");
		try {
			const stats = fs.statSync(GO_APP_LOCATION);
			logInfo("Binary exists:", stats.isFile());
			logInfo("Binary size:", stats.size);
			logInfo("Binary permissions:", stats.mode.toString(8));
		} catch (err) {
			logErr("Binary not found at path:", GO_APP_LOCATION);
			logErr("Error checking binary:", err);

			// List what's actually in the directory
			const dir = join(process.cwd(), waveConfig.Core.DistDir);
			try {
				const files = fs.readdirSync(dir);
				logInfo("Files in dist directory:", files);
			} catch (dirErr) {
				logErr("Cannot read dist directory:", dir, dirErr);
			}

			// Check parent directories
			logInfo("Checking parent directories:");
			const path = await import("node:path");
			let currentPath = process.cwd();
			for (let i = 0; i < 3; i++) {
				try {
					const contents = fs.readdirSync(currentPath);
					logInfo(
						`Contents of ${currentPath}:`,
						contents.slice(0, 10),
					); // First 10 items
				} catch {
					logErr(`Cannot read ${currentPath}`);
				}
				currentPath = path.dirname(currentPath);
			}
		}

		logInfo("Spawning Go process with PORT:", PORT);
		goProcess = spawn(GO_APP_LOCATION, [], {
			env: { ...process.env, PORT: PORT.toString() },
			stdio: "pipe",
		});

		goProcess.stdout?.on("data", (data) =>
			console.log(`[Go]: ${data.toString().trim()}`),
		);
		goProcess.stderr?.on("data", (data) =>
			console.error(`[Go ERR]: ${data.toString().trim()}`),
		);

		goProcess.on("exit", (code, signal) => {
			logInfo(`Go process exited with code ${code}, signal ${signal}.`);
			goProcess = null;
		});

		const healthCheckURL = `http://localhost:${PORT}${GO_APP_HEALTH_CHECK_ENDPOINT}`;
		await waitForGoApp(healthCheckURL, GO_APP_STARTUP_TIMEOUT_IN_MS);

		goReady = true;
		logInfo(
			`Go app is ready in ${(performance.now() - startTime).toFixed(2)}ms.`,
		);
	} catch (err) {
		logErr("Fatal error during initialization:", err);
		goProcess?.kill();
		goProcess = null;
		throw err;
	}
}

const startupPromise = init();
await startupPromise;

export default async function handler(req: VercelRequest, res: VercelResponse) {
	try {
		if (!goReady) {
			logInfo("Go app is not ready yet, waiting for initialization.");
			await startupPromise;
		}

		if (!goProcess || goProcess.killed) {
			res.status(503).send(
				"Service Unavailable: The backend service is not running.",
			);
			return;
		}

		const proxyReq = http.request(
			{
				hostname: "localhost",
				port: PORT,
				path: req.url,
				method: req.method,
				headers: req.headers,
			},
			(proxyRes) => {
				res.writeHead(proxyRes.statusCode ?? 500, proxyRes.headers);
				proxyRes.pipe(res, { end: true });
			},
		);

		proxyReq.on("error", (err) => {
			logErr("Error proxying request:", err);
			if (!res.headersSent) {
				res.status(502).send("Bad Gateway");
			}
			res.end();
		});

		req.pipe(proxyReq, { end: true });
	} catch (error) {
		logErr("Handler error:", error);
		if (!res.headersSent) {
			res.status(500).send(
				"Internal Server Error: Initialization failed.",
			);
		}
	}
}

process.on("SIGTERM", () => {
	if (goProcess) {
		logInfo("SIGTERM received, shutting down Go process.");
		goProcess.kill("SIGTERM");
	}
});

function logInfo(message?: any, ...optionalParams: any[]) {
	console.log(`[Node Proxy]: ${message}`, ...optionalParams);
}
function logErr(message?: any, ...optionalParams: any[]) {
	console.error(`[Node Proxy ERR]: ${message}`, ...optionalParams);
}
