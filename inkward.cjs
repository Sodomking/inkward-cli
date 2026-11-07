#!/usr/bin/env node
'use strict';
const {spawn}=require('node:child_process');
const readline=require('node:readline/promises');
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto');
const RELEASE_SITE = 'https://rimbaud.asia/writing/';
const VERSION='1.1.0';
const EXAMS=['english1','postgraduate','cet4','cet6','ielts','ielts-general'];
const TRIALS={english1:{big:'english1-big-2026',small:'english1-small-2026'},postgraduate:{big:'postgraduate-english2-2026',small:'postgraduate-english2-small-2026'}};
const RULE='DOMAIN-SUFFIX,rimbaud.asia,DIRECT';
const MARKER='# INKWARD direct rule v1';
function options(args){
  const out={command:'info',yes:false,json:false,dev:false,timeout:8000,format:'clash'};let command=false;
  for(let i=0;i<args.length;i++){
    const a=args[i];
    if(['info','open','setup','doctor','network','help','version'].includes(a)){if(command)throw Error('一次只能运行一个命令');out.command=a;command=true;}
    else if(['--json','--yes','--dev','--apply','--undo','--dry-run'].includes(a))out[a.slice(2)]=true;
    else if(['--site','--exam','--ref','--trial','--config','--format','--timeout'].includes(a)){if(!args[i+1]||args[i+1].startsWith('--'))throw Error(a+' 缺少参数');out[a.slice(2)]=args[++i];}
    else throw Error('不支持的参数：'+a);
  }
  if(out.exam==='english2')out.exam='postgraduate';
  if(out.exam==='ielts-academic')out.exam='ielts';
  if(out.exam&&!EXAMS.includes(out.exam))throw Error('考试参数无效');
  if(out.ref&&!/^[a-zA-Z0-9_-]{1,40}$/.test(out.ref))throw Error('邀请码格式无效');
  if(out.trial&&(!['big','small'].includes(out.trial)||!TRIALS[out.exam||'english1']))throw Error('免费试题请选择英语一或英语二的 big / small');
  if(!['clash','clash-verge','surge'].includes(out.format))throw Error('规则格式无效');
  out.timeout=Number(out.timeout);if(!Number.isInteger(out.timeout)||out.timeout<1000||out.timeout>30000)throw Error('超时范围为 1000–30000 毫秒');
  if(out.apply&&out.undo)throw Error('不能同时添加和撤销规则');
  if((out.apply||out.undo||out.config||out['dry-run'])&&out.command!=='network')throw Error('配置修改仅用于 network 命令');
  if((out.apply||out.undo)&&!out.config)throw Error('请用 --config 指定自己的自定义规则文件');
  if(out.config&&!out.apply&&!out.undo)throw Error('请明确使用 --apply 或 --undo；可加 --dry-run 先查看');
  return out;
}
function baseURL(o={}){
  const raw=o.site||RELEASE_SITE;
  if(typeof raw!=='string'||raw!==raw.trim()||/[\\\s%]/.test(raw)||raw.split('/').some(x=>x==='.'||x==='..'))throw Error('网站地址格式无效');
  const u=new URL(raw);
  if(u.username||u.password||u.search||u.hash||!/^\/(?:[A-Za-z0-9_-]+\/)*[A-Za-z0-9_-]*\/?$/.test(u.pathname))throw Error('网站地址不能含账户、查询参数或特殊路径');
  if(u.protocol!=='https:'&&!(o.dev&&u.protocol==='http:'&&['127.0.0.1','localhost'].includes(u.hostname)))throw Error('仅接受 HTTPS 网站；本机调试需 --dev');
  if(!u.pathname.endsWith('/'))u.pathname+='/';return u;
}
function website(o={}){
  const u=baseURL(o);if(o.ref)u.searchParams.set('ref',o.ref);
  u.hash=o.trial?'/brief?topic='+TRIALS[o.exam||'english1'][o.trial]:o.exam?'/exams?exam='+o.exam:'/home';return u.href;
}
function launchPlan(url,platform=process.platform){
  if(platform==='win32'){const script="Start-Process -FilePath '"+url.replaceAll("'","''")+"'";return {command:'powershell.exe',args:['-NoProfile','-NonInteractive','-EncodedCommand',Buffer.from(script,'utf16le').toString('base64')]};}
  return {command:platform==='darwin'?'open':'xdg-open',args:[url]};
}
function launch(url){
  const p=launchPlan(url);
  return new Promise((resolve,reject)=>{const child=spawn(p.command,p.args,{shell:false,stdio:'ignore'});child.once('error',()=>reject(Error('无法启动系统浏览器，请复制网址打开。')));child.once('exit',code=>code===0?resolve():reject(Error('浏览器启动失败，请复制网址打开。')));});
}
// Explicit native agents do not inherit HTTP_PROXY. OS VPN/TUN routing can still apply.
function request(url,{timeout=8000,method='GET',limit=256000}={}){
  return new Promise((resolve,reject)=>{
    const transport=require(new URL(url).protocol==='http:'?'node:http':'node:https');
    const agent=new transport.Agent({keepAlive:false,proxyEnv:{}});
    let done=false,timer;
    const finish=(error,value)=>{if(done)return;done=true;clearTimeout(timer);agent.destroy();error?reject(error):resolve(value);};
    const req=transport.request(url,{method,agent,headers:{'User-Agent':'INKWARD-CLI/'+VERSION,'Accept-Encoding':'identity'}},res=>{
      let bytes=0;const chunks=[];
      res.on('data',chunk=>{bytes+=chunk.length;if(bytes>limit){const e=Error('Response too large');e.code='response_too_large';finish(e);req.destroy();}else chunks.push(chunk);});
      res.on('error',e=>finish(e));res.on('end',()=>finish(null,{status:res.statusCode,text:Buffer.concat(chunks).toString('utf8')}));
    });
    timer=setTimeout(()=>{const e=Error('Timeout');e.code='timeout';finish(e);req.destroy();},timeout);
    req.on('error',e=>finish(e));req.end();
  });
}
async function diagnose(o,read=request,env=process.env){
  const base=baseURL(o),checks=[];
  async function probe(name,url,method='GET'){
    const start=Date.now();try{const r=await read(url,{timeout:o.timeout,method});const check={name,url,status:r.status,ok:r.status>=200&&r.status<300,ms:Date.now()-start};checks.push(check);return {...r,check};}
    catch(e){checks.push({name,url,ok:false,code:/^[A-Za-z0-9_]+$/.test(e.code||'')?e.code:'connection_failed',ms:Date.now()-start});return {};}
  }
  const [page,product]=await Promise.all([probe('website',base.href),probe('api',new URL('api/product',base).href)]);
  if(product.check){let valid=false;try{valid=JSON.parse(product.text).name==='INKWARD';}catch{}product.check.ok&&=valid;if(!valid)product.check.code='unexpected_product';}
  const candidates=[...(page.text||'').matchAll(/(?:src|href)=["'](https:\/\/static\.rimbaud\.asia\/releases\/[a-f0-9]{16}\/[^"']+)["']/g)];
  if(candidates.length){const cdn=new URL(candidates[0][1]);if(cdn.origin==='https://static.rimbaud.asia')await probe('cdn',cdn.href,'HEAD');}
  else checks.push({name:'cdn',ok:false,code:'not_found_in_page'});
  const proxyVariables=['HTTP_PROXY','HTTPS_PROXY','ALL_PROXY','http_proxy','https_proxy','all_proxy'].filter(k=>!!env[k]);
  return {ok:checks.every(c=>c.ok),website:base.href,transport:'direct-http-agent',browserRoutingVerified:false,proxyVariables,checks,
    guidance:['本检查不使用终端 HTTP 代理，但不能绕过系统 TUN/VPN。浏览器仍使用自己的网络设置。','若检查通过而网页仍慢，检查客户端 Rule 分流或安全验证连接；运行 network 查看本站直连规则。']};
}
function networkInfo(format='clash'){
  return {domain:'rimbaud.asia',format,rule:RULE,snippet:format==='surge'?'[Rule]\n'+RULE+'\n':format==='clash-verge'?'prepend:\n  - '+RULE+'\n':'rules:\n  - '+RULE+'\n',
    changed:false,activation:'not-applied',guidance:'将规则置于自定义规则前部，启用 Rule 模式并重载配置。仅本站及子域名直连；不修改全局代理或第三方验证码域名。'};
}
function editRules(source,{format='clash',undo=false}={}){
  if(format==='surge')throw Error('Surge 请导入输出的规则；自动编辑仅支持 Clash/Mihomo 自定义列表。');
  const newline=source.includes('\r\n')?'\r\n':'\n';const lines=source.split(/\r?\n/),key=format==='clash-verge'?'prepend':'rules';
  const markers=lines.map((line,i)=>line.trim()===MARKER?i:-1).filter(i=>i>=0);
  const starts=lines.map((line,i)=>new RegExp('^'+key+':\\s*(?:#.*)?$').test(line)?i:-1).filter(i=>i>=0);
  if(starts.length!==1||lines.some(l=>/^\s*(---|\.\.\.)\s*$/.test(l))||lines.some(l=>new RegExp('^["\x27]?'+key+'["\x27]?:').test(l)&&!new RegExp('^'+key+':\\s*(?:#.*)?$').test(l)))throw Error('只自动编辑单文档、顶层 '+key+': 的普通列表；内联数组或别名请手动导入规则。');
  const start=starts[0];let end=start+1;
  while(end<lines.length&&(!lines[end].trim()||/^\s*#/.test(lines[end])||/^\s/.test(lines[end])||/^-(?:\s|$)/.test(lines[end])))end++;
  const items=lines.slice(start+1,end).filter(l=>l.trim()&&!/^\s*#/.test(l));
  if(items.some(l=>!/^\s*-\s+(?:["']?[A-Z][A-Z0-9-]*,)/.test(l)))throw Error('列表包含复杂 YAML；请在客户端自定义规则中添加。');
  if(markers.length){
    if(markers.length!==1||markers[0]!==start+1||lines[markers[0]+1]?.trim()!=='- '+RULE)throw Error('已添加规则被移动或更改；为保留你的设置，请在客户端检查。');
    if(!undo)return {text:source,changed:false};
    lines.splice(markers[0],2);return {text:lines.join(newline),changed:true};
  }
  if(undo)return {text:source,changed:false};
  const indent=items.length?items[0].match(/^\s*/)[0]:'  ';
  lines.splice(start+1,0,indent+MARKER,indent+'- '+RULE);return {text:lines.join(newline),changed:true};
}
function updateRules(o){
  const file=path.resolve(o.config),stat=fs.lstatSync(file);
  if(!stat.isFile()||stat.isSymbolicLink()||stat.nlink!==1||stat.size>2*1024*1024)throw Error('规则文件须为普通文件，不能是链接或超过 2 MB。');
  const before=fs.readFileSync(file),source=before.toString('utf8');
  if(!Buffer.from(source).equals(before))throw Error('规则文件须为 UTF-8。');
  const edit=editRules(source,o);
  const result={...networkInfo(o.format),file,changed:false,wouldChange:edit.changed,operation:o.undo?'undo':'apply'};
  if(o['dry-run']||!edit.changed)return result;
  const stamp=Date.now()+'-'+crypto.randomBytes(4).toString('hex'),backup=file+'.inkward-'+stamp+'.bak',temp=file+'.inkward-'+stamp+'.tmp';
  // Private backups can contain subscription credentials. Never print their contents.
  fs.writeFileSync(backup,before,{flag:'wx',mode:0o600});
  try{
    fs.writeFileSync(temp,edit.text,{flag:'wx',mode:stat.mode&0o777});
    const current=fs.lstatSync(file);
    if(current.isSymbolicLink()||current.ino!==stat.ino||!fs.readFileSync(file).equals(before))throw Error('规则文件同时发生变化，未覆盖。');
    fs.renameSync(temp,file);
  }finally{if(fs.existsSync(temp))fs.unlinkSync(temp);}
  return {...result,changed:true,backup,activation:'reload-required',guidance:'文件已更新。请在代理客户端重载此自定义配置并确认 Rule 模式；尚未验证客户端实际采用此规则。'};
}
async function confirm(o,message){
  if(o.yes)return true;
  if(!process.stdin.isTTY)throw Error('该操作需明确授权；用户已要求时可加 --yes。');
  const rl=readline.createInterface({input:process.stdin,output:process.stderr});try{return /^y(es)?$/i.test((await rl.question(message+' [y/N] ')).trim());}finally{rl.close();}
}
function print(o,value,text){console.log(o.json?JSON.stringify(value,null,2):text||JSON.stringify(value,null,2));}
async function main(args=process.argv.slice(2),deps={}){
  const o=options(args);
  if(o.command==='version'){console.log(VERSION);return;}
  if(o.command==='help'){console.log(`inkward info [--json]\ninkward open [--exam ${EXAMS.join('|')}] [--trial big|small] [--yes]\ninkward setup [--exam english1] [--yes] [--json]\ninkward doctor [--json] [--timeout 8000]\ninkward network [--format clash|clash-verge|surge] [--json]\ninkward network --apply|--undo --config /path/to/custom-rules.yaml [--dry-run] [--yes]\n可选：--site https://example.com/writing/；本地调试加 --dev`);return;}
  if(o.command==='network'){
    if(!o.config){const result=networkInfo(o.format);print(o,result,result.snippet+'\n'+result.guidance);return;}
    if(!o['dry-run']&&!await confirm(o,'将备份并'+(o.undo?'撤销本站规则：':'添加本站直连规则：')+path.resolve(o.config)))return;
    const result=updateRules(o);print(o,result);return result;
  }
  const url=website(o);
  if(o.command==='info'){
    const info={name:'INKWARD',version:VERSION,website:url,exams:EXAMS,trials:TRIALS,description:'英语应试写作：核心表达、主动回忆、句子衔接、材料迁移与范文。',plans:[{id:'postgraduate',name:'考研'},{id:'cet',name:'四六级'},{id:'ielts',name:'雅思'}].map(p=>({...p,priceCNY:49,periodDays:30})),automaticRenewal:false,
      account:'登录、支付和学习记录在官网完成；CLI 不读取密码、Cookie、支付凭据或 API 密钥。',actions:{setup:'inkward setup --yes',doctor:'inkward doctor --json',network:'inkward network --json'},limitations:['未承诺固定天数提分','CLI 无法强制所有浏览器或 TUN/VPN 绕过代理','浏览器启动成功不代表已登录或网页已加载']};
    print(o,info,`INKWARD ${VERSION}\n${info.description}\n${url}\n三个套餐分别 49 元 / 30 天，手动续费；活动以官网为准。\n${info.account}`);return info;
  }
  let diagnosis;
  if(o.command==='doctor'||o.command==='setup'){
    diagnosis=await diagnose(o,deps.request||request,deps.env||process.env);
    if(o.command==='doctor'){print(o,diagnosis);if(!diagnosis.ok)process.exitCode=2;return diagnosis;}
  }
  if(!await confirm(o,'将用系统浏览器打开 '+url))return;
  await (deps.launch||launch)(url);
  const result={ok:true,website:url,browserRequested:true,pageLoadVerified:false,...(diagnosis?{diagnosis}:{}),next:'浏览器中继续试用或登录。需要直连时先运行 network 查看方案。'};
  print(o,result,`已请求系统浏览器打开：${url}${diagnosis?'\n网络诊断：'+(diagnosis.ok?'本站、API、CDN 均可连接。':'存在连接问题，请运行 doctor --json 查看。'):''}\n需要本站直连：inkward network`);return result;
}
if(require.main===module)main().catch(e=>{if(process.argv.includes('--json'))console.log(JSON.stringify({ok:false,error:e.message}));else console.error(e.message);process.exitCode=1;});
module.exports={options,baseURL,website,launchPlan,diagnose,request,networkInfo,editRules,updateRules,main};
