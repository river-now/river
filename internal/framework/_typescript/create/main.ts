#!/usr/bin/env node

import {
	cancel,
	confirm,
	intro,
	isCancel,
	log,
	select,
	spinner,
	text,
} from "@clack/prompts";
import { execSync } from "node:child_process";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

async function main() {
	intro("🌊 Create River App");

	// Check Go installation
	try {
		const goVersion = execSync("go version", { encoding: "utf8" }).trim();
		const versionMatch = goVersion.match(/go(\d+)\.(\d+)/);
		if (versionMatch) {
			if (!versionMatch[1] || !versionMatch[2]) {
				cancel(
					"Go version not recognized. Please ensure Go is installed correctly. See https://go.dev/doc/install for installation instructions.",
				);
				process.exit(1);
			}
			const major = parseInt(versionMatch[1]);
			const minor = parseInt(versionMatch[2]);
			if (major < 1 || (major === 1 && minor < 24)) {
				cancel("Go version 1.24 or higher is required");
				process.exit(1);
			}
		}
	} catch {
		cancel("Go is not installed. Please install Go 1.24 or higher");
		process.exit(1);
	}

	// Check Node version
	const nodeVersion = process.version;
	const firstPart = nodeVersion.split(".")[0];
	if (!firstPart || firstPart.length < 2 || !firstPart.startsWith("v")) {
		cancel(
			"Node.js version not recognized. Please ensure Node.js is installed correctly.",
		);
		process.exit(1);
	}
	const nodeMajor = parseInt(firstPart.substring(1));
	if (nodeMajor < 22) {
		cancel("Node.js version 22.11 or higher is required");
		process.exit(1);
	}

	// Option to create a new directory at start if not already in desired location
	const createNewDir = await confirm({
		message: "Create a new directory for your River app?",
		initialValue: false,
	});

	if (isCancel(createNewDir)) {
		cancel("Operation cancelled");
		process.exit(0);
	}

	let targetDir = process.cwd();

	if (createNewDir) {
		const dirName = await text({
			message: "Enter directory name:",
			placeholder: "my-river-app",
			validate: (value) => {
				if (!value || value.trim() === "")
					return "Directory name is required";
				// Check for invalid characters in directory name
				if (!/^[a-zA-Z0-9-_]+$/.test(value)) {
					return "Directory name can only contain letters, numbers, hyphens, and underscores";
				}
				// Check if directory already exists
				const proposedPath = path.join(process.cwd(), value);
				if (fs.existsSync(proposedPath)) {
					return `Directory "${value}" already exists`;
				}
				return undefined;
			},
		});

		if (isCancel(dirName)) {
			cancel("Operation cancelled");
			process.exit(0);
		}

		// Create the directory and change to it
		targetDir = path.join(process.cwd(), dirName as string);
		fs.mkdirSync(targetDir, { recursive: true });
		process.chdir(targetDir);
		log.success(`Created directory: ${dirName}`);
	}

	// Find go.mod and determine import path
	let goModPath: string | null = null;
	let moduleRoot: string | null = null;
	let moduleName: string | null = null;
	let createNewModule = false;

	// Search for go.mod
	let currentDir = process.cwd();
	while (currentDir !== path.dirname(currentDir)) {
		const modPath = path.join(currentDir, "go.mod");
		if (fs.existsSync(modPath)) {
			goModPath = modPath;
			moduleRoot = currentDir;
			break;
		}
		currentDir = path.dirname(currentDir);
	}

	// Parse module name if found
	if (goModPath) {
		const modContent = fs.readFileSync(goModPath, "utf8");
		const moduleMatch = modContent.match(/^module\s+(.+)$/m);
		if (moduleMatch && moduleMatch[1]) {
			moduleName = moduleMatch[1].trim();
		}

		// When module is found, give option to use it or create nested module
		const moduleChoice = await select({
			message: `Found parent Go module: ${moduleName}`,
			options: [
				{ value: "use", label: "Use existing module" },
				{ value: "new", label: "Create new nested module" },
			],
		});

		if (isCancel(moduleChoice)) {
			cancel("Operation cancelled");
			process.exit(0);
		}

		if (moduleChoice === "new") {
			createNewModule = true;
			// Reset module info since we're creating a new one
			goModPath = null;
			moduleRoot = null;
			moduleName = null;
		}
	} else {
		// No module found, we need to create one
		createNewModule = true;
	}

	// Handle go.mod initialization if needed
	if (createNewModule) {
		const modNameInput = await text({
			message:
				'Enter module name (e.g., "myapp" or "github.com/user/myapp"):',
			validate: (value) => {
				if (!value || value.trim() === "")
					return "Module name is required";
				return undefined;
			},
		});

		if (isCancel(modNameInput)) {
			cancel("Operation cancelled");
			process.exit(0);
		}

		moduleName = modNameInput as string;
		moduleRoot = process.cwd();

		const s = spinner();
		s.start("Initializing Go module");
		try {
			execSync(`go mod init ${moduleName}`, { cwd: moduleRoot });
			s.stop("Go module initialized");
		} catch (error) {
			s.stop("Failed to initialize module");
			cancel(`Error: ${error}`);
			process.exit(1);
		}
	}

	// Check for underscore directories from module root to current directory
	const pathFromModuleRoot = moduleRoot
		? path.relative(moduleRoot, process.cwd())
		: "";
	const pathSegments = pathFromModuleRoot
		.split(path.sep)
		.filter((seg) => seg.length > 0);
	const underscoreSegments = pathSegments.filter((seg) =>
		seg.startsWith("_"),
	);
	if (underscoreSegments.length > 0) {
		cancel(
			`Cannot create River app in a path containing directories that start with underscores:\n   ${underscoreSegments.join(", ")}\n` +
				`   Go ignores directories starting with underscores, which will cause build issues.`,
		);
		process.exit(1);
	}

	// Calculate import path automatically
	let importPath = moduleName!;
	if (moduleRoot !== process.cwd()) {
		const relativePath = path.relative(moduleRoot!, process.cwd());
		importPath = path.posix.join(
			moduleName!,
			...relativePath.split(path.sep),
		);
	}

	// Collect options
	const uiVariant = await select({
		message: "Choose UI framework:",
		options: [
			{ value: "react", label: "React" },
			{ value: "solid", label: "Solid" },
			{ value: "preact", label: "Preact" },
		],
	});

	if (isCancel(uiVariant)) {
		cancel("Operation cancelled");
		process.exit(0);
	}

	const packageManager = (await select({
		message: "Choose JS package manager:",
		options: [
			{ value: "npm", label: "npm" },
			{ value: "pnpm", label: "pnpm" },
			{ value: "yarn", label: "yarn" },
			{ value: "bun", label: "bun" },
		],
	})) as string;

	if (isCancel(packageManager)) {
		cancel("Operation cancelled");
		process.exit(0);
	}

	const deploymentTarget = await select({
		message: "Choose deployment target:",
		options: [
			{ value: "generic", label: "Generic (anywhere)" },
			{
				value: "vercel",
				label: "Vercel (adds some Vercel-specific config)",
			},
		],
	});

	if (isCancel(deploymentTarget)) {
		cancel("Operation cancelled");
		process.exit(0);
	}

	// Ask about Tailwind CSS
	const includeTailwind = await confirm({
		message: "Include Tailwind CSS?",
		initialValue: true,
	});

	if (isCancel(includeTailwind)) {
		cancel("Operation cancelled");
		process.exit(0);
	}

	// Show feedback that the process has started after all prompts are done
	log.info(`\n📦 Setting up your River app with:`);
	log.message(`   • UI Framework: ${uiVariant}`);
	log.message(`   • Package Manager: ${packageManager}`);
	log.message(`   • Deployment: ${deploymentTarget}`);
	log.message(`   • Tailwind CSS: ${includeTailwind ? "Yes" : "No"}`);
	log.message(`   • Module: ${moduleName}`);
	if (createNewDir) {
		log.message(`   • Location: ${targetDir}`);
	}
	log.info(`\n🚀 Starting setup process...\n`);

	// Create temporary directory
	const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "create-river-"));
	const bootstrapFile = path.join(tempDir, "main.go");

	try {
		// Write bootstrap Go file
		const goCode = `package main

import "github.com/river-now/river/bootstrap"

func main() {
	bootstrap.Init(bootstrap.Options{
		GoImportBase:     "${importPath}",
		UIVariant:        "${uiVariant}",
		JSPackageManager: "${packageManager}",
		DeploymentTarget: "${deploymentTarget}",
		IncludeTailwind:  ${includeTailwind},
	})
}
`;
		fs.writeFileSync(bootstrapFile, goCode);

		// Install River dependency
		const packageJsonPath = path.join(__dirname, "../package.json");
		const packageJson = JSON.parse(
			fs.readFileSync(packageJsonPath, "utf8"),
		);
		const version = packageJson.version;
		const s1 = spinner();
		s1.start("Installing River dependency");
		try {
			execSync(`go get github.com/river-now/river@v${version}`, {
				cwd: process.cwd(),
				stdio: "pipe",
			});
			s1.stop("River dependency installed");
		} catch (error) {
			s1.stop("Failed to install River");
			throw error;
		}

		// Run bootstrap
		const s2 = spinner();
		s2.start("Creating River app structure");
		try {
			execSync(`go run ${bootstrapFile}`, {
				cwd: process.cwd(),
				stdio: "pipe", // Changed from "inherit" to "pipe" to use spinner
			});
			s2.stop("River app created successfully!");
		} catch (error) {
			s2.stop("Failed to create app");
			throw error;
		}

		// Success message
		log.success(`\n✨ Your River app is ready!`);
		if (createNewDir) {
			log.info(`\n📁 Get started with:`);
			log.message(`   cd ${path.basename(targetDir)}`);
			log.message(`   ${packageManager} run dev`);
		} else {
			log.info(`\n📁 Get started with:`);
			log.message(`   ${packageManager} run dev`);
		}
	} catch (error) {
		cancel(`Error: ${error}`);
		process.exit(1);
	} finally {
		// Clean up temp directory
		try {
			fs.rmSync(tempDir, { recursive: true, force: true });
		} catch {
			// Ignore cleanup errors
		}
	}
}

main().catch((error) => {
	console.error("Unexpected error:", error);
	process.exit(1);
});
