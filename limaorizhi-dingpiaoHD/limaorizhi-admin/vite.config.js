import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import AutoImport from 'unplugin-auto-import/vite';
import Components from 'unplugin-vue-components/vite';
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers';
import path from 'path';
export default defineConfig({
    plugins: [
        vue(),
        AutoImport({
            resolvers: [ElementPlusResolver()],
        }),
        Components({
            resolvers: [ElementPlusResolver()],
        }),
    ],
    resolve: {
        alias: {
            '@': path.resolve(__dirname, 'src'),
        },
    },
    server: {
        port: 3000,
        proxy: {
            '/admin': {
                target: 'http://localhost:8181',
                changeOrigin: true,
            },
            '/uploads': {
                target: 'http://localhost:8181',
                changeOrigin: true,
            },
            '/api': {
                target: 'http://localhost:8181',
                changeOrigin: true,
            },
        },
    },
});
