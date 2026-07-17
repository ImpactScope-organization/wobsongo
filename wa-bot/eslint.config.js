// eslint.config.js
import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import promise from 'eslint-plugin-promise';
import prettierConfig from 'eslint-config-prettier';

export default tseslint.config(
  js.configs.recommended,
  ...tseslint.configs.recommendedTypeChecked,
  promise.configs['flat/recommended'],
  {
    languageOptions: {
      parserOptions: {
        project: './tsconfig.json',
        tsconfigRootDir: import.meta.dirname,
      },
    },
    rules: {
      '@typescript-eslint/no-floating-promises': 'error',
      '@typescript-eslint/no-misused-promises': 'error',
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],

      '@typescript-eslint/no-explicit-any': 'warn',

      'no-empty-function': 'off',
      '@typescript-eslint/no-empty-function': 'warn',
    },
  },
  prettierConfig, 
  {
    ignores: ['dist/**', 'node_modules/**', 'auth_info_baileys/**'],
  }
);