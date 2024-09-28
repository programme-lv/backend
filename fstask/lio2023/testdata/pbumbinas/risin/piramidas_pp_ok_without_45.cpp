#include <bits/stdc++.h>

using namespace std;
using ll = long long;

const int MAXC = 1000000;
const int MAXH = 1000000000;
const int MAXV = MAXH + MAXC + 10;

const int MAXST = 250 * 1024 * 1024 / (4 * 10);
struct ST {
    int l, r;
    int lc, rc;
    int stop_l, stop_r;
    int startval, prog; 
    ll sval;
}s[MAXST];

int cntT = 0;
int root;

int init(int l, int r)
{
    // cout << "INIT " << l << " " << r << endl;
    assert(l <= r);
    int it = ++cntT;
    assert(it < MAXST);
    s[it] = ST {
        .l = l,
        .r = r,
        .lc = 0,
        .rc = 0,
        .stop_l = (r - l + 1), 
        .stop_r = (r - l + 1),
        .startval = 0,
        .prog = 0,
        .sval = 0,
    };

    return it;
}

struct PosV {
    int h;
    int pos;
};

void extend(int it)
{
    int mid = s[it].l + (s[it].r - s[it].l) / 2;
    s[it].lc = init(s[it].l, mid);
    s[it].rc = init(mid + 1, s[it].r);
}

void push_part(int pl, int start, int prog, int it)
{
    s[it].startval = start + (prog * (s[it].l - pl));
    s[it].prog = prog;
    if (prog > 0)
    {
        s[it].stop_l = 0;
        s[it].stop_r = 2 * (s[it].r - s[it].l + 1);
    }
    else
    {
        s[it].stop_l = 2 * (s[it].r - s[it].l + 1);
        s[it].stop_r = 0;
    }
    s[it].sval = (s[it].startval + s[it].startval + prog * (ll)(s[it].r - s[it].l)) * (s[it].r - s[it].l + 1) / 2;
}

void push(int it)
{
    if (s[it].lc == 0)
    {
        extend(it);
    }
    if (s[it].prog == 0)
    {
        return;
    }
    push_part(s[it].l, s[it].startval, s[it].prog, s[it].lc);
    push_part(s[it].l, s[it].startval, s[it].prog, s[it].rc);
    s[it].prog = 0;
}

void update(int it)
{
    s[it].stop_r = s[s[it].lc].stop_r + s[s[it].rc].stop_r;
    s[it].stop_l = s[s[it].lc].stop_l + s[s[it].rc].stop_l;
    s[it].sval = s[s[it].lc].sval + s[s[it].rc].sval;
}

ll get_sum(int it, int l, int r)
{
    if (s[it].sval == 0)
    {
        return 0;
    }

    if (s[it].r < l || r < s[it].l)
    {
        return 0;
    }

    if (l <= s[it].l && s[it].r <= r)
    {
        return s[it].sval;
    }

    if (s[it].prog != 0)
    {
        // fast return
        int delta = max(l - s[it].l, 0);
        int startval = s[it].startval + delta * s[it].prog;
        int from = s[it].l + delta;
        int end = min(r, s[it].r);
        assert(from <= end);
        return (startval + startval + (ll)(end - from) * s[it].prog) * (ll)(end - from + 1) / 2;
    }

    push(it);
    ll sum = 0;
    sum += get_sum(s[it].lc, l, r);
    sum += get_sum(s[it].rc, l, r);
    return sum;
}

int get_pos_stopr_inner(int it, int pos, int& remdelta)
{
    if (s[it].r < pos || s[it].stop_r == 0)
    {
        return 0;
    }

    if (s[it].l == s[it].r)
    {
        remdelta -= s[it].stop_r;
        return s[it].l;
    }

    if (pos <= s[it].l)
    {
        if (s[it].stop_r < remdelta)
        {
            remdelta -= s[it].stop_r;
            return 0;
        }

        // Fast return
        if (s[it].prog == 1)
        {
            // stop_l > remdelta // correct direction
            int out = s[it].l + (remdelta - 1) / 2;
            remdelta = 0;
            return out;
        }

        if (s[it].sval == 0 && (remdelta - 1) <= (s[it].r - s[it].l))
        {
            int out = s[it].l + (remdelta - 1);
            remdelta = 0;
            return out;
        }
    }

    push(it);
    int out_pos = get_pos_stopr_inner(s[it].lc, pos, remdelta);
    if  (remdelta <= 0)
    {
        return out_pos;
    }
    return get_pos_stopr_inner(s[it].rc, pos, remdelta);
}

int get_pos_stopr(int it, int pos, int remdelta)
{
    return get_pos_stopr_inner(it, pos, remdelta);
}

int get_pos_stopl_inner(int it, int pos, int& remdelta)
{
    if (pos < s[it].l || s[it].stop_l == 0)
    {
        return 0;
    }

    if (s[it].l == s[it].r)
    {
        remdelta -= s[it].stop_l;
        return s[it].l;
    }

    if (s[it].r <= pos)
    {
        if (s[it].stop_l < remdelta)
        {
            remdelta -= s[it].stop_l;
            return 0;
        }

        // Fast return
        if (s[it].prog == -1)
        {
            // stop_l > remdelta // correct direction
            int out = s[it].r - (remdelta - 1) / 2;
            remdelta = 0;
            return out;
        }

        if (s[it].sval == 0 && (remdelta - 1) <= (s[it].r - s[it].l))
        {
            int out = s[it].r - (remdelta - 1);
            remdelta = 0;
            return out;
        }
    }

    push(it);
    int out_pos = get_pos_stopl_inner(s[it].rc, pos, remdelta);
    if  (remdelta <= 0)
    {
        return out_pos;
    }
    return get_pos_stopl_inner(s[it].lc, pos, remdelta);
}

int get_pos_stopl(int it, int pos, int remdelta)
{
    return get_pos_stopl_inner(it, pos, remdelta);
}

int get_val(int it, int pos)
{
    if (s[it].l == s[it].r)
    {
        return s[it].sval;
    }
    push(it);
    if (pos <= s[s[it].lc].r)
        return get_val(s[it].lc, pos);
    return get_val(s[it].rc, pos);
}

ll filled_pyramid_ballcnt(int pos, int h, int val)
{
    int delta = h - val;
    assert(delta >= 0);
    // Uz katru pusi aizpildisim tris rindas

    int lpos = get_pos_stopl(root, pos, delta);
    int rpos = get_pos_stopr(root, pos, delta);
    ll total_sum = get_sum(root, lpos, rpos);

    int dl = pos - lpos;
    int dr = rpos - pos;
    assert(dr >= 0);

    ll total_left_mid = (h - dl + h) * (ll)(dl + 1) / 2;
    ll total_right = (h - dr + h - 1) * (ll)dr / 2;

    ll total_expected = total_left_mid + total_right;

    return total_expected - total_sum;
}

pair<int, ll> binsearch_pyramid(int pos, ll ballcnt, int val)
{
    int L = val + 1;
    int R = MAXH;

    int bestansw = val;
    ll bestballcnt = 0;

    while (L <= R)
    {
        int mid = L + (R - L) / 2;
        ll found_ballcnt = filled_pyramid_ballcnt(pos, mid, val);
        if (found_ballcnt <= ballcnt)
        {
            L = mid + 1;
            bestansw = mid;
            bestballcnt = found_ballcnt;
        }
        else
        {
            R = mid - 1;
        }
    }

    return {bestansw, bestballcnt};
}

void set_prog_inner(int it, int l, int r, int spos, int start_val, int prog)
{
    if (s[it].r < l)
        return;
    if (r < s[it].l)
        return;

    if (l <= s[it].l && s[it].r <= r)
    {
        push_part(spos, start_val, prog, it);
        return;
    }

    push(it);
    set_prog_inner(s[it].lc, l, r, spos, start_val, prog);
    set_prog_inner(s[it].rc, l, r, spos, start_val, prog);
    update(it);
}

int update_pair_val[2];
void update_pair_tree(int it, int pos)
{
    if (s[it].r < pos || pos + 1 < s[it].l)
        return;

    if (s[it].l == s[it].r)
    {
        int off = s[it].l - pos;
        if (off == 0)
        {
            if (update_pair_val[0] < update_pair_val[1])
            {
                s[it].stop_r = 2;
            }
            else if (update_pair_val[0] == update_pair_val[1])
            {
                s[it].stop_r = 1;
            }
            else
            {
                s[it].stop_r = 0;
            }
        }
        else if (off == 1)
        {
            if (update_pair_val[0] > update_pair_val[1])
            {
                s[it].stop_l = 2;
            }
            else if (update_pair_val[0] == update_pair_val[1])
            {
                s[it].stop_l = 1;
            }
            else
            {
                s[it].stop_l = 0;
            }
        }
        return;
    }

    push(it);
    update_pair_tree(s[it].lc, pos);
    update_pair_tree(s[it].rc, pos);
    update(it);
}

void update_pair(int pos)
{
    update_pair_val[0] = get_val(root, pos);
    update_pair_val[1] = get_val(root, pos + 1);
    update_pair_tree(root, pos);
}

void set_prog(int l, int r, int prog, int startval)
{
    if (r < l)
        return;
    set_prog_inner(root, l, r, l, startval, prog);
    update_pair(l - 1);
    update_pair(r);
}

void fill_pyramid(int pos, int newh, int h)
{
    int delta = newh - h;
    if (delta == 0)
    {
        return;
    }
    assert(delta >= 0);
    // Uz katru pusi aizpildisim tris rindas

    int lpos = get_pos_stopl(root, pos, delta);
    int rpos = get_pos_stopr(root, pos, delta);

    set_prog(lpos, pos, 1, newh - (pos - lpos));
    set_prog(pos + 1, rpos, -1, newh - 1);
}

pair<int, ll> binsearch_and_fill_pyramid(int pos, ll ballcnt)
{
    int val = get_val(root, pos);
    pair<int, ll> answ = binsearch_pyramid(pos, ballcnt, val);
    fill_pyramid(pos, answ.first, val);
    return answ;
}

int main()
{
    root = init(-MAXV, MAXV);

    int n;
    scanf("%d", &n);

    for (; n > 0; n--)
    {
        int pos;
        ll ballcnt;
        scanf("%d %lld", &pos, &ballcnt);

        pair<int, ll> answ = binsearch_and_fill_pyramid(pos, ballcnt);
        // cout << "FOUND " << answ.first << " " << answ.second << endl;

        PosV answpos;
        if (ballcnt == answ.second)
        {
            answpos = {
                .h = answ.first,
                .pos = pos
            };
        }
        else
        {
            ballcnt -= answ.second;
            assert(ballcnt > 0);

            int posr = get_pos_stopr(root, pos, 1);
            if (posr - pos >= ballcnt)
            {
                int out_pos = posr - ballcnt + 1;
                answpos = {
                    .h = answ.first + 1 - (out_pos - pos),
                    .pos = out_pos,
                };
                set_prog(out_pos, posr, -1, answpos.h);
            }
            else
            {
                ballcnt -= posr - pos;
                set_prog(pos + 1, posr, -1, answ.first);
                int posl = get_pos_stopl(root, pos, 1);
                assert(pos - posl + 1 >= ballcnt);

                int out_pos = posl + ballcnt - 1;
                set_prog(posl, out_pos, 1, answ.first + 1 - (pos - posl));
                answpos = {
                    .h = answ.first + 1 - (pos - out_pos),
                    .pos = out_pos
                };
            }
        }

        printf("%d %d\n", answpos.pos, answpos.h);
    }
}
