// VimQuest canvas 2D 렌더러 — Go game.go 의 snapshot() 계약만 읽는다.
// 게임 규칙은 전혀 모른다(듀얼 프론트엔드 드리프트 방지) — main.go(Ebiten)의
// drawPlaying/drawLevelClear/drawLevelSelect/drawAllClear 와 기능 동등.
'use strict';

const CW = 14, LH = 28, W = 960, H = 600;
// 이 팔레트가 canonical — cmd/desktop/render.go 의 col* 상수와 index.html 의
// CSS 변수(3벌째)는 여기 값을 손으로 맞춘 사본이다.
const COL = {
  bg: '#1e202a', floor: '#4a4f5e', key: '#f4d03f', keyDim: '#6a6030',
  pest: '#e54b4b', exit: '#4fc36b', cursor: '#3aa0d0', ins: '#f4d03f',
  visual: '#554a80', text: '#e8e8e8', muted: '#8a90a0', match: '#355540',
};

class Renderer {
  constructor(canvas) {
    this.canvas = canvas;
    // HiDPI: 캔버스의 물리 픽셀(backing store)만 devicePixelRatio 배로 키우고
    // ctx.scale 로 상쇄한다 — 이후 모든 draw* 호출은 여전히 논리 좌표계
    // (W×H = 960×600)로 그리면 되고, 레티나에서만 자동으로 선명해진다.
    const dpr = window.devicePixelRatio || 1;
    canvas.width = Math.round(W * dpr);
    canvas.height = Math.round(H * dpr);
    this.ctx = canvas.getContext('2d');
    this.ctx.scale(dpr, dpr);
    // basicfont 의 투박한 느낌을 유지하되 신규 에셋은 쓰지 않는다(시스템 monospace).
    this.ctx.font = '26px Menlo, Consolas, monospace';
    this.ctx.textBaseline = 'top';

    this.applyResponsiveScale();
    window.addEventListener('resize', () => this.applyResponsiveScale());
  }

  // applyResponsiveScale 은 논리 캔버스(960×600)의 표시 크기(CSS 픽셀 — 위의
  // 물리 픽셀 backing store 와는 별개)를 뷰포트에 맞게 줄인다. 정수 배율
  // 후보(1, 0.75, 0.5) 중 뷰포트에 맞는 가장 큰 값만 골라 적용 — 임의
  // 소수 배율은 쓰지 않는다(CSS image-rendering:pixelated 와 함께 정수
  // 배율에서만 격자가 깨끗하게 유지된다).
  applyResponsiveScale() {
    const scales = [1, 0.75, 0.5];
    const avail = document.documentElement.clientWidth || window.innerWidth;
    let chosen = scales[scales.length - 1];
    for (const s of scales) {
      // #panel/#ime-warning 등 기존 max-width:96vw 관례와 같은 여유폭.
      if (W * s <= avail * 0.96) { chosen = s; break; }
    }
    this.canvas.style.width = `${Math.round(W * chosen)}px`;
    this.canvas.style.height = `${Math.round(H * chosen)}px`;
  }

  // draw 는 상태에 맞는 draw* 로 분기하는 단일 진입점(Go 의 Game.Draw 와 대응).
  draw(st) {
    switch (st.state) {
      case 'allclear': return this.drawAllClear(st);
      case 'clear': return this.drawLevelClear(st);
      case 'select': return this.drawLevelSelect(st);
      case 'drillSummary': return this.drawDrillSummary(st);
      case 'arenaDone': return this.drawArenaDone(st);
      default: return this.drawPlaying(st); // playing / drill / arena
    }
  }

  clear() {
    this.ctx.fillStyle = COL.bg;
    this.ctx.fillRect(0, 0, W, H);
  }

  ch(s, x, y, col) {
    this.ctx.fillStyle = col;
    this.ctx.fillText(s, x, y);
  }

  rect(x, y, w, h, col) {
    this.ctx.fillStyle = col;
    this.ctx.fillRect(x, y, w, h);
  }

  cellColor(c, kind, hasKey) {
    if (kind !== 'navigate') return COL.text;
    switch (c) {
      case 'K': return hasKey ? COL.key : COL.keyDim;
      case '*': return COL.pest;
      case '$': return COL.exit;
      case '.': return COL.floor;
      default: return COL.text;
    }
  }

  // hasKey 는 (r,c) 에 아직 안 주운 열쇠가 있는지(snapshot 의 keyPos 배열 기준).
  hasKey(st, r, c) {
    return !!(st.keyPos && st.keyPos.some(p => p.row === r && p.col === c));
  }

  effectAt(st, r, c) {
    return (st.effects || []).find(e => e.row === r && e.col === c);
  }

  drawPlaying(st) {
    this.clear();
    // st.title 이 생성기별 유형을 이미 담고 있다("DRILL"/"DRILL [w]"/...).
    // 아레나는 문제 번호를 크게 — 라이브 타이머는 캔버스가 아니라
    // #arena-panel(DOM)이 소유한다(시간은 JS 몫이라는 경계 유지).
    let hud = st.arena
      ? `ARENA  problem ${st.arena.num}/${st.arena.count}`
      : st.drill
        ? `${st.title}   streak ${st.drill.streak}`
        : `level ${st.level}/${st.levelCount}`;
    hud += st.kind === 'navigate'
      ? `   keys ${st.keys}/${st.keysNeed}   bugs ${st.bugs}`
      : '   [EDIT]  transform LEFT to match RIGHT';
    this.ch(hud, 60, 50, COL.muted);

    const oy = 130;
    if (st.kind === 'navigate') {
      this.drawBuffer(st, 60, oy, null);
    } else {
      this.ch('CURRENT', 60, oy - 26, COL.text);
      this.ch('TARGET', 540, oy - 26, COL.exit);
      this.drawBuffer(st, 60, oy, st.target);
      this.drawTarget(st.target, 540, oy);
      this.rect(510, oy - 10, 2, 300, COL.floor);
    }

    let bar = st.exMode
      ? ':' + st.exBuf
      : `${st.mode}   cmd: ${st.pending}   last: ${st.last}   keys ${st.strokes} / par ${st.par}`;
    if (!st.exMode && st.drill) {
      bar += `   total ${st.drill.totalKeys}/${st.drill.totalPar}`;
    }
    if (st.bell) {
      this.rect(40, H - 52, W - 80, 32, COL.text);
      this.ch(bar, 60, H - 46, COL.bg);
    } else {
      this.ch(bar, 60, H - 46, COL.text);
    }
  }

  drawBuffer(st, ox, oy, target) {
    // 비주얼 선택 구간·목표줄 일치 여부는 게임이 계산해 내려준다
    // (visualRows/matchedRows) — 렌더러는 읽기만 한다.
    const visByRow = new Map((st.visualRows || []).map(v => [v.row, v]));
    const matched = st.matchedRows || [];
    (st.lines || []).forEach((line, r) => {
      if (target && matched[r]) {
        this.rect(ox - 2, oy + r * LH - 2, Math.max(line.length, 1) * CW + 4, LH - 2, COL.match);
      }
      const vr = visByRow.get(r);
      for (let c = 0; c < line.length; c++) {
        const px = ox + c * CW, py = oy + r * LH;
        if (vr && c >= vr.c1 && c <= vr.c2) this.rect(px - 1, py - 2, CW, LH - 4, COL.visual);
        if (r === st.row && c === st.col) {
          this.rect(px - 1, py - 2, CW, LH - 4, st.mode.includes('INSERT') ? COL.ins : COL.cursor);
        }
        let g = line[c];
        let col = this.cellColor(g, st.kind, this.hasKey(st, r, c));
        const eff = this.effectAt(st, r, c);
        if (eff) {
          if (eff.invert) { this.rect(px - 1, py - 2, CW, LH - 4, COL.text); col = COL.bg; }
          if (eff.glyph) { g = eff.glyph; col = COL.pest; }
        }
        if (g !== ' ') this.ch(g, px, py, col);
      }
      if (r === st.row && st.col >= line.length) {
        const cc = st.mode.includes('INSERT') ? COL.ins : COL.cursor;
        this.rect(ox + line.length * CW - 1, oy + r * LH - 2, CW, LH - 4, cc);
      }
    });
  }

  drawTarget(lines, ox, oy) {
    (lines || []).forEach((line, r) => {
      for (let c = 0; c < line.length; c++) {
        if (line[c] !== ' ') this.ch(line[c], ox + c * CW, oy + r * LH, COL.exit);
      }
    });
  }

  drawLevelClear(st) {
    this.clear();
    this.ch(`LEVEL ${st.id} CLEAR!`, 340, 220, COL.exit);
    this.ch(`your keys : ${st.clearStrokes}`, 340, 260, COL.text);
    const stars = '*'.repeat(st.clearStars) + '-'.repeat(3 - st.clearStars);
    this.ch(`par       : ${st.clearPar}   ${stars}`, 340, 290, COL.text);
    let best = `best      : ${st.clearBest}`;
    if (st.clearIsNew) best += ` -> ${st.clearStrokes} (NEW!)`;
    this.ch(best, 340, 320, COL.muted);
    // 내가 실제로 입력한 키 시퀀스 — 별점과 무관하게 항상 표시(제작자 solution
    // 과 달리 스포일러가 아니다). COPY 버튼(glue.js)이 이 값을 읽는다.
    this.ch(`yours     : ${st.clearYours || ''}`, 340, 350, COL.text);
    if (st.clearStars === 3 && st.solution) this.ch(`solution  : ${st.solution}`, 340, 380, COL.key);
    this.ch('[Enter] next   [r] retry', 340, 420, COL.muted);
  }

  // drawDrillSummary 는 :drill 세션을 ":q"/":levels" 로 빠져나올 때의 통계
  // 요약 화면 — drillStreak/drillTotalKeys/drillTotalPar 는 세션 내내
  // 누적돼 온 값을 그대로 읽는다(요약 화면에서 다시 계산할 게 없다).
  drawDrillSummary(st) {
    this.clear();
    const pct = st.drillTotalPar > 0 ? Math.round(st.drillTotalKeys * 100 / st.drillTotalPar) : 0;
    this.ch('DRILL SESSION SUMMARY', 340, 220, COL.exit);
    this.ch(`streak    : ${st.drillStreak}`, 340, 260, COL.text);
    this.ch(`keys/par  : ${st.drillTotalKeys}/${st.drillTotalPar} (${pct}%)`, 340, 290, COL.text);
    this.ch('[any key] back to level select', 340, 330, COL.muted);
  }

  // drawArenaDone — 최종 기록·제출·리더보드는 전부 #arena-panel(DOM)에
  // 있으므로 캔버스는 완주 안내만 띄운다.
  drawArenaDone(st) {
    this.clear();
    this.ch('ARENA COMPLETE!', 340, 250, COL.exit);
    this.ch(`all ${st.arenaCount} problems solved.`, 340, 290, COL.text);
    this.ch('submit your time in the ARENA panel above', 340, 330, COL.muted);
  }

  drawLevelSelect(st) {
    this.clear();
    this.ch('SELECT LEVEL', 60, 50, COL.text);
    this.ch('h/l world   j/k level   Enter play   Esc back', 60, 80, COL.muted);
    (st.worlds || []).forEach((group, wi) => {
      const ox = 60 + wi * 220;
      this.ch(`W${wi + 1}`, ox, 130, COL.exit);
      group.forEach((lv, li) => {
        const oy = 170 + li * 36;
        if (wi === st.selRow && li === st.selCol) this.rect(ox - 4, oy - 2, 196, 24, COL.visual);
        const label = lv.unlocked
          ? `${lv.id} ${'*'.repeat(lv.stars)}${'-'.repeat(3 - lv.stars)}`
          : `${lv.id} LOCK`;
        this.ch(label, ox, oy, lv.unlocked ? COL.text : COL.muted);
      });
    });
  }

  drawAllClear(st) {
    this.clear();
    this.ch('ALL CLEAR!', 360, 250, COL.exit);
    this.ch(`W1-W${st.worldCount} ${st.levelCount} levels complete.`, 300, 290, COL.text);
    this.ch('[Enter] level select', 340, 330, COL.muted);
  }
}
