import resolve from '@rollup/plugin-node-resolve';
import commonjs from '@rollup/plugin-commonjs';
import typescript from '@rollup/plugin-typescript';
import terser from '@rollup/plugin-terser';

export default {
  input: 'src/index.ts',
  output: [
    {
      file: 'dist/lm-forms.js',
      format: 'iife',
      name: 'Listmonk',
      sourcemap: true,
    },
    {
      file: 'dist/lm-forms.min.js',
      format: 'iife',
      name: 'Listmonk',
      plugins: [terser()],
      sourcemap: true,
    },
    {
      file: 'dist/lm-forms.esm.js',
      format: 'es',
      sourcemap: true,
    },
  ],
  plugins: [
    resolve(),
    commonjs(),
    typescript({ tsconfig: './tsconfig.json' }),
  ],
};
