import react from '@vitejs/plugin-react-swc';
import { resolve } from 'path';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [react()],
  base: '/admin/static/form-builder/',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    minify: 'terser',
    rollupOptions: {
      output: {
        manualChunks: undefined,
        entryFileNames: 'form-builder.js',
        chunkFileNames: 'form-builder-[hash].js',
        assetFileNames: 'form-builder-[hash][extname]',
      },
    },
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
});
