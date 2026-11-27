const {chromium}=require('playwright');
const assert=require('node:assert/strict');
const http=require('node:http');
const fs=require('node:fs');

(async()=>{
 const html=fs.readFileSync('static/index.html');
 const server=http.createServer((req,res)=>{
  if(req.url==='/static/board.png') {res.writeHead(404);res.end();return;}
  res.setHeader('Content-Type','text/html; charset=utf-8');res.end(html);
 });
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
 const browser=await chromium.launch({...(process.env.CHROME_PATH?{executablePath:process.env.CHROME_PATH}:{}),headless:true});
 try {
  const page=await browser.newPage({viewport:{width:390,height:844}});
  const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.route('https://fonts.googleapis.com/**',r=>r.abort());
  await page.addInitScript(()=>{
   window.sockets=[];window.sent=[];
   window.mockState={phase:'lobby',code:'TEST',round:0,revision:0,host:true,myID:'h',canStart:false,players:[{id:'h',name:'Eduardo',host:true,connected:true}]};
   window.WebSocket=class {
    static OPEN=1;static CONNECTING=0;
    constructor() {this.readyState=0;window.sockets.push(this);setTimeout(()=>{this.readyState=1;this.onopen?.();},0);}
    send(raw) {
     const msg=JSON.parse(raw);window.sent.push(msg);
     if(msg.type==='join') {
      this.emit({type:'joined',code:'TEST',host:true,name:'Eduardo',token:'test-session-token'});
      this.emit(window.mockState);
     }
     if(msg.type==='ping') this.emit({type:'pong'});
    }
    emit(d) {this.onmessage?.({data:JSON.stringify(d)});}
    close() {if(this.readyState===3)return;this.readyState=3;setTimeout(()=>this.onclose?.(),0);}
   };
  });
  const url=`http://127.0.0.1:${server.address().port}`;
  await page.goto(url);
  await page.locator('#inputName').fill('Eduardo');
  await page.locator('#btnCreate').click();
  await page.waitForSelector('#lobby.active');
  assert.equal(await page.evaluate(()=>JSON.parse(localStorage.getItem('tier-du-session-v1')).token),'test-session-token');
  await page.evaluate(()=>window.sockets.at(-1).close());
  await page.waitForSelector('#connectionStatus:not([hidden])');
  assert.equal(await page.locator('#lobby').evaluate(el=>el.classList.contains('active')),true);
  await page.waitForFunction(()=>window.sent.filter(m=>m.type==='join').length===2);
  assert.equal(await page.evaluate(()=>window.sent.at(-1).data.token),'test-session-token');
  assert.equal(await page.locator('#connectionStatus').isHidden(),true);
  await page.reload();
  await page.waitForSelector('#lobby.active');
  await page.waitForFunction(()=>window.sent.some(m=>m.type==='join'));
  assert.equal(await page.evaluate(()=>window.sent[0].data.token),'test-session-token');
  await page.evaluate(()=>{
   window.mockState={...window.mockState,phase:'answering',round:1,revision:1,currentTheme:'Sabores de pizza',myTier:'E',myLeaderName:'Ana',themeVotes:0,themeVotesNeeded:2,timerEnd:Date.now()+90000};
   window.sockets.at(-1).emit(window.mockState);
  });
  assert.match(await page.locator('#timer').innerText(),/^1:(30|29)$/);
  assert.equal(await page.locator('.theme-heading .theme-label').innerText(),'TEMA DA RODADA');
  assert.equal(await page.locator('.theme-heading h1').innerText(),'Sabores de pizza');
  assert.equal(await page.locator('.board-theme').innerText(),'Tema da vez: Sabores de pizza');
  await page.locator('#answerText').fill('Rascunho salvo ao sair do navegador');
  await page.reload();
  // Mock the same live round after reload; the saved session must recover its draft.
  await page.evaluate(()=>{
   window.mockState={...window.mockState,phase:'answering',round:1,revision:1,currentTheme:'Sabores de pizza',myTier:'E',myLeaderName:'Ana',themeVotes:0,themeVotesNeeded:2,timerEnd:Date.now()+90000};
   if(window.sockets.at(-1).readyState===1)window.sockets.at(-1).emit(window.mockState);
  });
  await page.waitForSelector('#answerText');
  assert.equal(await page.locator('#answerText').inputValue(),'Rascunho salvo ao sair do navegador');
  await page.evaluate(()=>{lastMessageAt=0;document.dispatchEvent(new Event('visibilitychange'));});
  await page.waitForFunction(()=>window.sent.filter(m=>m.type==='join').length===2);
  assert.equal(await page.locator('#answerText').inputValue(),'Rascunho salvo ao sair do navegador');
  assert.equal(await page.locator('#connectionStatus').isHidden(),true);
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);
  await page.evaluate(()=>{window.sockets.at(-1).emit({...window.mockState,revision:2,currentTheme:'Bebidas'});});
  assert.equal(await page.locator('#answerText').inputValue(),'');
  await page.evaluate(()=>window.sockets.at(-1).emit({type:'sessionReplaced'}));
  assert.equal(await page.evaluate(()=>session===null),true);
  assert.notEqual(await page.evaluate(()=>localStorage.getItem('tier-du-session-v1')),null);
  assert.equal(await page.locator('#entry').evaluate(el=>el.classList.contains('active')),true);
  await page.evaluate(()=>{const s=JSON.parse(localStorage.getItem('tier-du-session-v1'));connect(s.name,s.code,s.token);});
  await page.waitForFunction(()=>session!==null && awaitingState===false);
  page.once('dialog',d=>d.accept());
  await page.evaluate(()=>window.sockets.at(-1).emit({error:'sessao_expirada'}));
  assert.equal(await page.evaluate(()=>localStorage.getItem('tier-du-session-v1')),null);
  assert.equal(await page.locator('#entry').evaluate(el=>el.classList.contains('active')),true);
  assert.deepEqual(errors,[]);
  console.log('PASS: saved lobby, automatic retry, same token, reload, 1:30 timer, both theme labels, draft recovery, background return, theme reset, tab handover, expired session, mobile layout.');
 } finally {await browser.close();await new Promise(resolve=>server.close(resolve));}
})().catch(e=>{console.error(e);process.exit(1);});
