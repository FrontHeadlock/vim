// VimQuest canvas 2D 렌더러 — Go game.go 의 snapshot() 계약만 읽는다.
// 게임 규칙은 전혀 모른다(듀얼 프론트엔드 드리프트 방지) — main.go(Ebiten)의
// drawPlaying/drawLevelClear/drawLevelSelect/drawAllClear 와 기능 동등.
'use strict';

const CW = 14, LH = 28, W = 960, H = 600;
const COL = {
  bg: '#1e202a', floor: '#4a4f5e', key: '#f4d03f', keyDim: '#6a6030',
  pest: '#e54b4b', exit: '#4fc36b', cursor: '#3aa0d0', ins: '#f4d03f',
  visual: '#554a80', text: '#e8e8e8', muted: '#8a90a0', match: '#355540',
};

class Renderer {
  constructor(canvas) {
    this.ctx = canvas.getContext('2d');
    // basicfont 의 투박한 느낌을 유지하되 신규 에셋은 쓰지 않는다(시스템 monospace).
    this.ctx.font = '26px Menlo, Consolas, monospace';
    this.ctx.textBaseline = 'top';
  }

  // draw 는 상태에 맞는 draw* 로 분기하는 단일 진입점(Go 의 Game.Draw 와 대응).
  draw(st) {
    switch (st.state) {
      case 'allclear': return this.drawAllClear(st);
      case 'clear': return this.drawLevelClear(st);
      case 'select': return this.drawLevelSelect(st);
      default: return this.drawPlaying(st);
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
    let hud = st.drill
      ? `DRILL   streak ${st.drill.streak}`
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
    const v = st.visual || { ok: false };
    (st.lines || []).forEach((line, r) => {
      if (target && r < target.length && line === target[r]) {
        this.rect(ox - 2, oy + r * LH - 2, Math.max(line.length, 1) * CW + 4, LH - 2, COL.match);
      }
      for (let c = 0; c < line.length; c++) {
        const px = ox + c * CW, py = oy + r * LH;
        if (v.ok && this.inVisual(r, c, v)) this.rect(px - 1, py - 2, CW, LH - 4, COL.visual);
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

  inVisual(r, c, v) {
    if (r < v.r1 || r > v.r2) return false;
    if (v.line) return true;
    if (v.r1 === v.r2) return c >= v.c1 && c <= v.c2;
    if (r === v.r1) return c >= v.c1;
    if (r === v.r2) return c <= v.c2;
    return true;
  }

  drawLevelClear(st) {
    this.clear();
    this.ch(`LEVEL ${st.id} CLEAR!`, 340, 220, COL.exit);
    this.ch(`your keys : ${st.clearStrokes}`, 340, 260, COL.text);
    const stars = '*'.repeat(st.clearStars) + '-'.repeat(3 - st.clearStars);
    this.ch(`par       : ${st.clearPar}   ${stars}`, 340, 290, COL.text);
    let best = `best      : ${st.clearBest}`;
    if (!st.clearBest || st.clearStrokes < st.clearBest) best += ` -> ${st.clearStrokes} (NEW!)`;
    this.ch(best, 340, 320, COL.muted);
    if (st.clearStars === 3 && st.solution) this.ch(`solution  : ${st.solution}`, 340, 350, COL.key);
    this.ch('[Enter] next   [r] retry', 340, 390, COL.muted);
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
    this.ch(`W1-W8 ${st.levelCount} levels complete.`, 300, 290, COL.text);
    this.ch('press the Restart button to replay', 250, 330, COL.muted);
  }
}
