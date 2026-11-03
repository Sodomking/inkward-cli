#!/usr/bin/env node
'use strict';
const {spawn}=require('node:child_process');
const readline=require('node:readline/promises');
// Set this during release build, after the production website passes its deployment checks.
const RELEASE_SITE = null;
const VERSION='1.0.0';
function options(args){
  const out={command:'info',yes:false,json:false,dev:false};
  for(let i=0;i<args.length;i++){
    const a=args[i];
    if(['info','open','doctor','help','version'].includes(a))out.command=a;
    else if(['--json','--yes','--dev'].includes(a))out[a.slice(2)]=true;
    else if(['--site','--exam','--ref'].includes(a)){if(!args[i+1]||args[i+1].startsWith('--'))throw new Error(a+' 缺少参数');out[a.slice(2)]=args[++i];}
    else throw new Error('不支持的参数：'+a);
  }
  if(out.exam&&!['ielts','postgraduate','cet6'].includes(out.exam))throw new Error('考试参数无效');
  if(out.ref&&!/^[a-zA-Z0-9_-]{1,40}$/.test(out.ref))throw new Error('邀请码格式无效');
  return out;
}
function website(o){
  const base=o.site||RELEASE_SITE;if(!base)throw new Error('网站域名尚未写入此版本。请使用 --site https://你的正式域名，或等待正式 Release。');
  const u=new URL(base);
  if(u.username||u.password||u.pathname!=='/'||u.search||u.hash)throw new Error('网站地址必须是不含路径、账户和参数的完整域名');
  if(u.protocol!=='https:'&&!(o.dev&&u.protocol==='http:'&&['127.0.0.1','localhost'].includes(u.hostname)))throw new Error('仅接受 HTTPS 网站；本机调试需 --dev');
  if(o.ref)u.searchParams.set('ref',o.ref);
  u.hash=o.exam?'/exams?exam='+o.exam:'/home';return u.href;
}
function launch(url){
  let command,args;
  if(process.platform==='win32'){command='powershell.exe';const script="Start-Process -FilePath '"+url.replaceAll("'","''")+"'";args=['-NoProfile','-NonInteractive','-EncodedCommand',Buffer.from(script,'utf16le').toString('base64')];}
  else if(process.platform==='darwin'){command='open';args=[url];}
  else {command='xdg-open';args=[url];}
  return new Promise((resolve,reject)=>{const p=spawn(command,args,{shell:false,stdio:'ignore'});p.on('error',reject);p.on('exit',code=>code===0?resolve():reject(new Error('未能自动打开浏览器，请复制上方网址。')));});
}
async function main(args=process.argv.slice(2)){
  const o=options(args);
  if(o.command==='version'){console.log(VERSION);return;}
  if(o.command==='help'){console.log('inkward info [--json]\ninkward open [--exam ielts|postgraduate|cet6] [--ref CODE] [--yes]\ninkward doctor\n可选：--site https://你的域名；本地调试 --site http://127.0.0.1:3000 --dev');return;}
  let url=null;try{url=website(o);}catch(e){if(o.command!=='info')throw e;}
  const info={name:'INKWARD',version:VERSION,description:'英语应试写作训练：词云、默写、选词、上下句写句与完整范文。',website:url,priceCNY:49,periodDays:30,automaticRenewal:false,
    account:'登录、微信支付和学习记录都在官方网站完成；CLI 不收集账号、密码、支付信息或 API 密钥。',actions:{open:'inkward open --yes',info:'inkward info --json'},limitations:['具体开放题库以网站显示为准','AI 需要联网和电脑操作权限才能代为运行','没有保证提分或固定天数学习效果']};
  if(o.command==='info'){console.log(o.json?JSON.stringify(info,null,2):`${info.name} ${VERSION}\n${info.description}\n49 元 / 30 天，手动续费。\n${url||'此构建尚未配置正式网站域名。'}\n${info.account}\n运行 inkward open 打开网站；inkward help 查看参数。`);return;}
  if(o.command==='doctor'){
    const origin=new URL(url).origin;
    try{const r=await fetch(origin+'/api/product',{signal:AbortSignal.timeout(8000),redirect:'error'});if(!r.ok)throw new Error();const p=await r.json();if(p.name!=='INKWARD')throw new Error();console.log(o.json?JSON.stringify({ok:true,website:origin,product:p}):'网站服务可连接：'+origin);}catch{throw new Error('网站未响应或尚未部署，请检查域名与网络。');}return;
  }
  console.log('将使用系统浏览器打开：'+url);
  if(!o.yes){if(!process.stdin.isTTY){console.log('自动化运行请在用户同意后加 --yes。');return;}const rl=readline.createInterface({input:process.stdin,output:process.stdout});const answer=await rl.question('是否打开？[y/N] ');rl.close();if(!/^y(es)?$/i.test(answer.trim()))return;}
  await launch(url);
}
if(require.main===module)main().catch(e=>{console.error(e.message);process.exitCode=1;});
module.exports={options,website,main};
