import { readFileSync, writeFileSync } from "node:fs";

const file = "public/assets/openpanel.js.LEGAL.txt";
writeFileSync(file, readFileSync(file, "utf8").replace(/[ \t]+$/gm, ""));
