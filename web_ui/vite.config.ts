import { defineConfig } from "vite"
import { viteSingleFile } from "vite-plugin-singlefile"
import { checker } from "vite-plugin-checker"


export default defineConfig(({ mode }) => {
	var server_cfg = {}
	if (mode == 'development') {
		var fazantix_url = process.env.FAZANTIX_URL
		if (!fazantix_url) {
			fazantix_url = 'http://localhost:8000'
		}
		server_cfg = {
			proxy: {
				'/api/ws': {
					target: fazantix_url.replace(/^http/, 'ws'),
					ws: true,
					changeOrigin: true,
					secure: false,
					rewrite: (path) => path,
				},
				'/api': {
					target: fazantix_url,
					changeOrigin: true,
					secure: false,
					rewrite: (path) => path,
				},
			},
		}
	}


	return {
		plugins: [
			viteSingleFile(),
			checker({ typescript: true }),
		],
		build: {
			assetsInlineLimit: Number.MAX_SAFE_INTEGER,
		},
		server: server_cfg
	}
})
