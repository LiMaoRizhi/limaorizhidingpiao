/* ============================================================
 * 狸猫日志售票系统 (limaorizhi Ticketing System)
 * Copyright (c) limaorizhi. All rights reserved.
 * 版权所有：狸猫日志 (limaorizhi)  保留所有权利
 * 项目：limaorizhi-admin  作者：limaorizhi
 * 未经授权不得复制、传播或用于商业用途
 * ============================================================ */
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
