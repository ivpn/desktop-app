import { defineConfig, externalizeDepsPlugin } from 'electron-vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// https://electron-vite.org/config/
export default defineConfig({
  //https://electron-vite.org/config/#built-in-config-for-main
  main: {    
    plugins: [externalizeDepsPlugin()],
    build: {
      lib: {
        entry:  resolve('./src/background.js'),        
      },      
    },   
    resolve: {
      alias: {
        "@": resolve("src")
      },
      extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json', '.vue']
    },     
  },
  // https://electron-vite.org/config/#built-in-config-for-preload
  preload: {
    plugins: [externalizeDepsPlugin()],
    build: {
      lib: {
        entry:  './src/preload.js',
      },      
    },  
    resolve: {
      alias: {
        "@": resolve("src")
      },
      extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json', '.vue']
    }, 
  },
  // https://electron-vite.org/config/#built-in-config-for-renderer
  renderer: {
    plugins: [vue()],    
    root: '.',
    build: {
      rollupOptions: {
        input: 'index.html'
      }
    },
    resolve: {
      alias: {
        "@": resolve("src")
      },
      extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json', '.vue']
    }, 
  }
})
