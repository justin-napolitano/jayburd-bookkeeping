// @vitest-environment node
import {expect,it} from 'vitest';
import {publicConfig} from './public-config';
import {parsePublicAuthConfig} from '../auth/config';
it('returns only the runtime public allowlist with no caching',async()=>{
 const response=publicConfig({JAYBURD_FIREBASE_API_KEY:'synthetic-public',JAYBURD_FIREBASE_PROJECT_ID:'synthetic-project',JAYBURD_FIREBASE_AUTH_DOMAIN:'synthetic-project.firebaseapp.com',UNRELATED_VALUE:'must-not-appear'});
 expect(response.status).toBe(200); expect(response.headers.get('Cache-Control')).toBe('no-store');
 expect(await response.json()).toEqual({apiKey:'synthetic-public',projectId:'synthetic-project',authDomain:'synthetic-project.firebaseapp.com'});
});
it('fails closed on partial, custom-host, malformed or extra config',async()=>{
 for(const env of [{},{JAYBURD_FIREBASE_API_KEY:'synthetic'},{JAYBURD_FIREBASE_API_KEY:'synthetic',JAYBURD_FIREBASE_PROJECT_ID:'synthetic-project',JAYBURD_FIREBASE_AUTH_DOMAIN:'evil.invalid'}]) expect(publicConfig(env).status).toBe(503);
 expect(parsePublicAuthConfig({apiKey:'synthetic',projectId:'synthetic-project',authDomain:'synthetic-project.firebaseapp.com',extra:'denied'})).toBeNull();
});
