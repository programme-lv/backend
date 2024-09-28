#include <bits/stdc++.h>

using namespace std;

using ll = long long;

const int MAXC = 1e6;
const int MAXV = 1e9;

map<int, int> pyramid_top;

struct Crosspoint {
  int pos, h;
  bool two;

  ll area() const {
    if (two) {
      return h * (ll)h + h;
    } else {
      return h * (ll)h;
    }
  }
};

Crosspoint crosspoint(int lpos, int lh, int rpos, int rh) {
  assert(lpos < rpos);
  assert(lpos + (rpos - lpos) <= rpos);

  if (lpos + lh - 1 < rpos - rh + 1) {
    return {lpos + lh, 0};
  }

  // Citādi krustojas

  int left_delta_h = rh - lh;
  lpos -= left_delta_h;
  lh += left_delta_h;

  if ((rpos - lpos) % 2 == 0) {
    int off = (rpos - lpos) / 2;
    return Crosspoint{lpos + off, lh - off, false};
  } else {
    int off = (rpos - lpos) / 2;
    return Crosspoint{lpos + off, lh - off - 1, true};
  }
}

ll calc_ball_cnt(int pos, int nexth, int prevh, int lpos, int lh, int rpos,
                 int rh) {
  if (nexth == prevh) {
    return 0;
  }
  ll prevballcnt = prevh * (ll)prevh;
  ll nextballcnt = nexth * (ll)nexth;
  ll delta = nextballcnt - prevballcnt;

  auto lcrosstop = crosspoint(lpos, lh, pos, nexth);
  auto lcrosslow = crosspoint(lpos, lh, pos, prevh);

  auto toplow = lcrosstop.area() - lcrosslow.area();
  delta -= toplow;

  auto rcrosstop = crosspoint(pos, nexth, rpos, rh);
  auto rcrosslow = crosspoint(pos, prevh, rpos, rh);
  toplow = rcrosstop.area() - rcrosslow.area();
  delta -= toplow;

  assert(delta >= 0);

  return delta;
}

pair<int, ll> binsearch_and_fill(int pos, ll ballcnt) {
  // Initialize
  int prevh = 0;
  auto rit = pyramid_top.lower_bound(pos);
  assert(rit != pyramid_top.begin());

  auto lit = rit;
  --lit;

  bool removed_self = false;

  if (rit->first == pos) {
    prevh = rit->second;
    rit++;
    removed_self = true;
  }

  assert(rit != pyramid_top.end());

  prevh = max(prevh, rit->second - (rit->first - pos));
  int overr = rit->second + (rit->first - pos);

  prevh = max(prevh, lit->second - (pos - lit->first));
  int overl = lit->second + (pos - lit->first);

  int initial_prevh = prevh;

  while (true) {
    // Check next height
    int nexth = min(overl, overr);
    assert(nexth > prevh);
    ll next_ball_cnt = calc_ball_cnt(pos, nexth, prevh, lit->first, lit->second,
                                     rit->first, rit->second);
    if (next_ball_cnt > ballcnt) {
      break;
    }

    ballcnt -= next_ball_cnt;
    if (overl == nexth) {
      assert(lit != pyramid_top.begin());
      lit--;
      overl = lit->second + (pos - lit->first);
    }
    if (overr == nexth) {
      rit++;
      overr = rit->second + (rit->first - pos);
    }
    prevh = nexth;
  }

  // Remove all eaten elements
  pyramid_top.erase(++lit, rit);

  rit = pyramid_top.lower_bound(pos);
  lit = rit;
  --lit;

  int R = min(overl, overr);
  int L = prevh + 1;

  while (L <= R) {
    int mid = L + (R - L) / 2;
    ll next_ball_cnt = calc_ball_cnt(pos, mid, prevh, lit->first, lit->second,
                                     rit->first, rit->second);
    if (next_ball_cnt <= ballcnt) {
      L = mid + 1;
    } else {
      R = mid - 1;
    }
  }

  int h = L - 1;
  if (h > initial_prevh || removed_self)
  {
    ll next_ball_cnt = calc_ball_cnt(pos, h, prevh, lit->first, lit->second,
                                     rit->first, rit->second);
    ballcnt -= next_ball_cnt;
    pyramid_top[pos] = h;
  }

  return {h, ballcnt};
}

bool is_under(int pos, int h, int npos, int nh) {
  return (nh + abs(npos - pos) <= h);
}

int main() {

  int n;
  scanf("%d", &n);

  // Fake pyramids that always will create a limit
  pyramid_top[-MAXC - MAXV - 9] = 1;
  pyramid_top[+MAXC + MAXV + 9] = 1;

  for (int i = 0; i < n; i++) {

    int pos;
    ll ballcnt;
    scanf("%d %lld", &pos, &ballcnt);

    auto output = binsearch_and_fill(pos, ballcnt);
    int h = output.first;
    ballcnt = output.second;
    if (ballcnt == 0) {
      printf("%d %d\n", pos, h);
      continue;
    }

    // Fill right side
    auto it = pyramid_top.lower_bound(pos);
    auto npos = it;
    if (it->first == pos)
    {
      ++npos;
    }

    auto cross = crosspoint(pos, h + 1, npos->first, npos->second);
    int stoph = cross.h;

    if (h - stoph >= ballcnt) {
      int tpos = pos + 1 + (h - stoph) - ballcnt;
      int th = stoph + ballcnt;
      if (is_under(tpos, th, npos->first, npos->second)) {
        pyramid_top.erase(npos);
      }
      pyramid_top[tpos] = th;
      printf("%d %d\n", tpos, th);
    } else {
      if (h - stoph > 0)
      {
        ballcnt -= h - stoph;
        int tpos = pos + 1;
        int th = h;
        if (is_under(tpos, th, npos->first, npos->second)) {
          pyramid_top.erase(npos);
        }
        pyramid_top[tpos] = th;
      }

      it = pyramid_top.lower_bound(pos);
      auto lpos = it;
      --lpos;
      auto cross = crosspoint(lpos->first, lpos->second, pos, h + 1);
      int stoph = cross.h;

      int tpos = pos - 1 - (h - stoph) + ballcnt;
      int th = stoph + ballcnt;
      if (is_under(tpos, th, lpos->first, lpos->second)) {
        pyramid_top.erase(lpos);
      }
      pyramid_top[tpos] = th;
      printf("%d %d\n", tpos, th);
    }
  }
}
