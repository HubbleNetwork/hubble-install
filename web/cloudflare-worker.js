// Cloudflare Worker for get.hubble.com
// Detects OS via User-Agent and serves appropriate installation script

const BASE = 'https://hubble-install.s3.amazonaws.com';

export default {
  async fetch(request) {
    const ua = request.headers.get('User-Agent') || '';
    const script = /windows|powershell/i.test(ua) ? 'install.ps1' : 'install.sh';
    
    const response = await fetch(`${BASE}/${script}`, {
      cf: { cacheTtl: 300, cacheEverything: true }
    });
    
    return new Response(response.body, {
      headers: {
        'Content-Type': 'text/plain; charset=utf-8',
        'Cache-Control': 'public, max-age=300'
      }
    });
  }
}
