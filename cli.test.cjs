const {test}=require('node:test'),assert=require('node:assert/strict');
const fs=require('node:fs'),os=require('node:os'),path=require('node:path');
const {options,website,baseURL,launchPlan,diagnose,request,editRules,updateRules,main}=require('./inkward.cjs');
test('official website, all exams and four complete trials',()=>{
 assert.equal(website({}),'https://rimbaud.asia/writing/#/home');
 for(const exam of ['english1','postgraduate','cet4','cet6','ielts','ielts-general'])assert.ok(website(options(['open','--exam',exam])).endsWith('/exams?exam='+exam));
 assert.equal(website(options(['open','--exam','english2','--trial','small'])),'https://rimbaud.asia/writing/#/brief?topic=postgraduate-english2-small-2026');
 assert.throws(()=>options(['open','--exam','cet4','--trial','big']));
});
test('safe subpaths and invitation encoding',()=>{
 assert.equal(website(options(['open','--site','https://writing.example.com/app','--ref','friends_01','--exam','postgraduate'])),'https://writing.example.com/app/?ref=friends_01#/exams?exam=postgraduate');
 for(const site of ['javascript:alert(1)','https://u:p@example.com','http://example.com','https://example.com/a/../b','https://example.com/?a=1','https://example.com/%2e','https://example.com/\\x','https://example.com/a b'])assert.throws(()=>website({site}),site);
 assert.equal(baseURL({site:'http://127.0.0.1:3000',dev:true}).pathname,'/');
});
test('no implicit execution or arbitrary shell flags',()=>{
 assert.equal(options([]).command,'info');assert.equal(options(['open']).yes,false);
 for(const args of [['--eval','bad'],['--ref','a&bad'],['--site'],['open','doctor'],['network','--apply'],['open','--config','file'],['--timeout','0']])assert.throws(()=>options(args));
});
test('cross-platform launch arguments never invoke a shell or interpolate URL as code',()=>{
 const url=website({});assert.deepEqual(launchPlan(url,'darwin'),{command:'open',args:[url]});assert.deepEqual(launchPlan(url,'linux'),{command:'xdg-open',args:[url]});
 const plan=launchPlan("https://example.com/a'b",'win32');assert.equal(plan.command,'powershell.exe');assert.equal(Buffer.from(plan.args.at(-1),'base64').toString('utf16le'),"Start-Process -FilePath 'https://example.com/a''b'");
});
const page='<script src="https://static.rimbaud.asia/releases/abcdef0123456789/app.js"></script>';
test('doctor uses application base path and checks CDN without exposing proxy secrets',async()=>{
 const calls=[];const r=await diagnose(options(['doctor']),async(url,o)=>{calls.push([url,o.method]);return {status:200,text:url.endsWith('api/product')?'{"name":"INKWARD"}':page};},{HTTPS_PROXY:'https://private:secret@proxy.invalid'});
 assert.equal(r.ok,true);assert.deepEqual(calls,[['https://rimbaud.asia/writing/','GET'],['https://rimbaud.asia/writing/api/product','GET'],['https://static.rimbaud.asia/releases/abcdef0123456789/app.js','HEAD']]);
 assert.deepEqual(r.proxyVariables,['HTTPS_PROXY']);assert.ok(!JSON.stringify(r).includes('secret'));assert.equal(r.browserRoutingVerified,false);
});
test('doctor rejects wrong API identity, external asset hosts and network errors',async()=>{
 const r=await diagnose(options(['doctor']),async url=>{if(url.endsWith('api/product'))throw Object.assign(Error('secret proxy URL'),{code:'ETIMEDOUT'});return {status:200,text:'<script src="https://attacker.invalid/a.js"></script>'};},{});
 assert.equal(r.ok,false);assert.equal(r.checks.find(c=>c.name==='cdn').code,'not_found_in_page');assert.ok(!JSON.stringify(r).includes('secret'));
 const wrong=await diagnose(options(['doctor']),async()=>({status:200,text:'{}'}),{});assert.equal(wrong.checks.find(c=>c.name==='api').ok,false);
});
test('native request bounds response size and total timeout',async t=>{
 const server=require('node:http').createServer((req,res)=>{if(req.url==='/large')res.end('a'.repeat(100));});await new Promise(r=>server.listen(0,'127.0.0.1',r));t.after(()=>server.close());
 const base='http://127.0.0.1:'+server.address().port;
 await assert.rejects(request(base+'/large',{limit:10}),{code:'response_too_large'});
 await assert.rejects(request(base+'/wait',{timeout:30}),{code:'timeout'});
});
test('rule is first, idempotent, reversible and preserves all other config bytes',()=>{
 for(const src of ['mode: rule\nrules:\n  - DOMAIN,example.com,PROXY\n  - MATCH,PROXY\n','rules:\r\n- MATCH,PROXY\r\n','rules: # custom\n']){
  const added=editRules(src);assert.equal(added.changed,true);assert.ok(added.text.indexOf('rimbaud.asia')<added.text.indexOf('MATCH')||!src.includes('MATCH'));
  assert.equal(editRules(added.text).changed,false);assert.equal(editRules(added.text,{undo:true}).text,src);
 }
 const verge='prepend:\n  - DOMAIN,example.com,DIRECT\n';assert.equal(editRules(editRules(verge,{format:'clash-verge'}).text,{format:'clash-verge',undo:true}).text,verge);
});
test('complex or changed config is refused without silent overwrite',()=>{
 for(const src of ['rules: []\n','rules: &my_rules\n  - MATCH,PROXY\n','rules:\nrules:\n','---\nrules:\n','rules:\n  nested: value\n'])assert.throws(()=>editRules(src));
 const modified=editRules('rules:\n  - MATCH,PROXY\n').text.replace('rimbaud.asia,DIRECT','rimbaud.asia,PROXY');assert.throws(()=>editRules(modified,{undo:true}));
});
test('file apply backs up, dry run does not write, undo preserves later unrelated edits',t=>{
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'inkward-test-'));t.after(()=>fs.rmSync(dir,{recursive:true,force:true}));
 const file=path.join(dir,'rules.yaml'),original='rules:\n  - MATCH,PROXY\n';fs.writeFileSync(file,original);
 assert.equal(updateRules({config:file,'dry-run':true}).wouldChange,true);assert.equal(fs.readFileSync(file,'utf8'),original);assert.equal(fs.readdirSync(dir).length,1);
 const added=updateRules({config:file});assert.equal(fs.readFileSync(added.backup,'utf8'),original);assert.equal(added.activation,'reload-required');assert.equal(updateRules({config:file}).changed,false);
 if(process.platform!=='win32')assert.equal(fs.statSync(added.backup).mode&0o777,0o600);
 fs.appendFileSync(file,'log-level: info\n');updateRules({config:file,undo:true});assert.equal(fs.readFileSync(file,'utf8'),original+'log-level: info\n');
});
test('symlink config cannot be overwritten',t=>{
 if(process.platform==='win32')return t.skip('Windows symlink creation may require privileges');
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'inkward-test-'));t.after(()=>fs.rmSync(dir,{recursive:true,force:true}));fs.writeFileSync(path.join(dir,'real'),'rules:\n');fs.symlinkSync(path.join(dir,'real'),path.join(dir,'link'));
 assert.throws(()=>updateRules({config:path.join(dir,'link')}));
});
test('setup runs diagnostics then requests correct browser URL; no login success claim',async()=>{
 let opened;const old=console.log;console.log=()=>{};try{
 const result=await main(['setup','--yes','--exam','english1','--trial','small','--json'],{request:async url=>({status:200,text:url.endsWith('api/product')?'{"name":"INKWARD"}':page}),launch:async url=>{opened=url;},env:{}});
 assert.equal(opened,website({exam:'english1',trial:'small'}));assert.equal(result.browserRequested,true);assert.equal(result.pageLoadVerified,false);
 }finally{console.log=old;}
});
test('no-Node starter matches CLI exam and trial destinations',()=>{
 const {execFileSync}=require('node:child_process');
 for(const exam of ['','english1','english2','cet4','cet6','ielts','ielts-general']){
  const args=process.platform==='win32'?['-NoProfile','-File',path.join(__dirname,'open-only.ps1'),'-PrintUrl',...(exam?['-Exam',exam]:[])]:[path.join(__dirname,'open-only.sh'),'--print-url',...(exam?['--exam',exam]:[])];
  const actual=execFileSync(process.platform==='win32'?'powershell.exe':'sh',args,{encoding:'utf8'}).trim();
  assert.equal(actual,website(options(['open',...(exam?['--exam',exam]:[])])));
 }
 for(const exam of ['english1','postgraduate'])for(const trial of ['big','small']){
  const args=process.platform==='win32'?['-NoProfile','-File',path.join(__dirname,'open-only.ps1'),'-PrintUrl','-Exam',exam,'-Trial',trial]:[path.join(__dirname,'open-only.sh'),'--print-url','--exam',exam,'--trial',trial];
  assert.equal(execFileSync(process.platform==='win32'?'powershell.exe':'sh',args,{encoding:'utf8'}).trim(),website({exam,trial}));
 }
});
