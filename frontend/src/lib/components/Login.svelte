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
        <div class="w-12 h-12 bg-indigo-600 rounded-xl flex items-center justify-center shadow-lg shadow-indigo-200">
          <svg class="w-7 h-7 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
        </div>
      </div>
      <h1 class="text-2xl font-bold text-slate-800">Welcome Back</h1>
      <p class="text-slate-500 mt-2">Sign in to access VectorGo</p>
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
