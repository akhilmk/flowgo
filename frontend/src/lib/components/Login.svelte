<script lang="ts">
  import { api } from "../api";

  let username = "";
  let password = "";
  let loading = false;
  let error = "";

  export let onLoginSuccess: () => void;

  async function handleLogin() {
    if (!username || !password) {
      error = "Please enter both username and password";
      return;
    }

    loading = true;
    error = "";

    try {
      await api.login(username, password);
      onLoginSuccess();
    } catch (err) {
      error = err instanceof Error ? err.message : "Login failed";
    } finally {
      loading = false;
    }
  }
</script>

<div class="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 to-indigo-50 p-4">
  <div class="max-w-md w-full bg-white rounded-2xl shadow-xl p-8 border border-white/50 backdrop-blur-sm">
    <div class="text-center mb-8">
      <div class="flex justify-center mb-4">
        <div class="w-16 h-16 bg-indigo-600 rounded-2xl flex items-center justify-center shadow-lg shadow-indigo-200 ring-4 ring-indigo-50">
          <svg class="w-10 h-10 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <!-- Database Cylinder Base -->
            <path d="M4 7v10c0 2.2 3.6 4 8 4s8-1.8 8-4V7"></path>
            <path d="M4 7c0 2.2 3.6 4 8 4s8-1.8 8-4-3.6-4-8-4-8 1.8-8 4z"></path>
            <path d="M4 12c0 2.2 3.6 4 8 4s8-1.8 8-4"></path>
            
            <!-- AI Sparkle/Neural Node Overlay -->
            <path d="M12 11v-3"></path>
            <circle cx="12" cy="7" r="1.5" fill="currentColor" stroke="none"></circle>
            
            <path d="M12 7l-4-2"></path>
            <circle cx="8" cy="5" r="1" fill="currentColor" stroke="none"></circle>
            
            <path d="M12 7l4-2"></path> 
            <circle cx="16" cy="5" r="1" fill="currentColor" stroke="none"></circle>
          </svg>
        </div>
      </div>
      <h1 class="text-3xl font-bold text-slate-900 tracking-tight">Gowise</h1>
      <p class="text-slate-500 mt-2 font-medium">AI-Powered Knowledge Base</p>
    </div>

    <form on:submit|preventDefault={handleLogin} class="space-y-6">
      <div>
        <label for="username" class="block text-sm font-semibold text-slate-700 mb-2">Username</label>
        <input
          id="username"
          type="text"
          bind:value={username}
          class="w-full px-4 py-3 rounded-lg border border-slate-300 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all"
          placeholder="Enter username"
          autocomplete="username"
        />
      </div>

      <div>
        <label for="password" class="block text-sm font-semibold text-slate-700 mb-2">Password</label>
        <input
          id="password"
          type="password"
          bind:value={password}
          class="w-full px-4 py-3 rounded-lg border border-slate-300 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all"
          placeholder="Enter password"
          autocomplete="current-password"
        />
      </div>

      {#if error}
        <div class="p-3 rounded-lg bg-red-50 border border-red-200 text-red-600 text-sm font-medium">
          {error}
        </div>
      {/if}

      <button
        type="submit"
        disabled={loading}
        class="w-full bg-indigo-600 text-white font-semibold py-3 px-6 rounded-lg hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-lg shadow-indigo-200"
      >
        {#if loading}
          <span class="flex items-center justify-center gap-2">
            <svg class="animate-spin h-5 w-5" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            Signing in...
          </span>
        {:else}
          Sign In
        {/if}
      </button>
    </form>
  </div>
</div>
