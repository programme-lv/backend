#include <bits/stdc++.h>

using namespace std;

using ll = long long;
using pint = int;

const int MAXC = 1000000;
const int SIDEC = 1000000000 + MAXC + 10;
const int MAXVS = 1000000000; // Due to segment tree restrictions

int n;
int root[2];

const int MAXST = 4000000;
struct STree {
    pint l, r;
    int lc, rc;
    pint pushv, max;
    ll total;
}s[MAXST];

int cntT = 0;

struct Pos45 {
    pint h45;
    pint pos45;
};

struct PosV {
    pint h;
    pint pos;
};

Pos45 cpos45(PosV val)
{
    pint pos45 = val.pos - val.h;

    return Pos45 {
        .h45 = pos45 / 2 + val.h,
        .pos45 = pos45,
    };
}

PosV cpos(Pos45 val)
{
    pint h = val.h45 - val.pos45 / 2;
    pint pos = val.pos45 + h;

    return PosV {
        .h = h,
        .pos = pos,
    };
}

void update(int it)
{
    assert(s[it].lc != 0 && s[it].pushv == 0);
    STree lc = s[s[it].lc];
    STree rc = s[s[it].rc];
    auto& cur = s[it];
    cur.total = lc.total + rc.total;
    cur.max = max(lc.max, rc.max);
}

void push_part(int cit, pint pushv)
{
    auto& c = s[cit];
    // c.max = max(c.max, pushv);
    c.max = pushv;
    c.total = (c.r - c.l + 1) * (ll)pushv;
    c.pushv = pushv;
}

int buildTree(pint l, pint r)
{
    int it = ++cntT;
    assert(it < MAXST);
    s[it] = {
        .l = l,
        .r = r,
        .lc = 0,
        .rc = 0,
        .pushv = -1,
        .max = r,
        .total = (l + (ll)r) * (r - l + 1) / 2,
    };
    return it;
}

void extendNode(int it)
{
    if (s[it].lc)
        return;
    pint mid = s[it].l + (s[it].r - s[it].l) / 2;
    s[it].lc = buildTree(s[it].l, mid);
    s[it].rc = buildTree(mid + 1, s[it].r);
    if (s[it].pushv == -1)
        s[it].pushv = 0;
}

void push(int it)
{
    extendNode(it);
    if (s[it].pushv <= 0)
        return;
    push_part(s[it].lc, s[it].pushv);
    push_part(s[it].rc, s[it].pushv);
    s[it].pushv = 0;
}


ll getSum(int it, pint l, pint r)
{
    if (l <= s[it].l && s[it].r <= r)
    {
        return s[it].total;
    }

    if (s[it].total == 0)
    {
        return 0;
    }
    
    if (s[it].pushv > 0)
    {
        pint from = max(s[it].l, l);
        pint to = min(s[it].r, r);
        if (from > to)
            return 0;
        return s[it].pushv * (ll)(to - from + 1);
    }

    if (s[it].pushv == -1)
    {
        pint from = max(s[it].l, l);
        pint to = min(s[it].r, r);
        assert(from <= to);
        return (from + to) * (ll)(to - from + 1) / 2;
    }

    push(it);
    ll answ = 0;
    if (l <= s[s[it].lc].r)
        answ += getSum(s[it].lc, l, r);
    if (s[s[it].rc].l <= r)
        answ += getSum(s[it].rc, l, r);
    return answ;
}

int getSumPos(int it, pint pos)
{
    return (int)getSum(it, pos, pos);
}

void setSum(int it, pint l, pint r, pint val)
{
    if (l <= s[it].l && s[it].r <= r)
    {
        assert(s[it].max <= val);
        s[it].pushv = val;
        s[it].total = val * (ll)(s[it].r - s[it].l + 1);
        assert(s[it].max <= val);
        s[it].max = val;
        return;
    }

    push(it);
    if (l <= s[s[it].lc].r)
        setSum(s[it].lc, l, r, val);
    if (s[s[it].rc].l <= r)
        setSum(s[it].rc, l, r, val);
    update(it);
}

int binsearch_ge(int it, pint val)
{
    if (s[it].l == s[it].r)
    {
        assert(s[it].max >= val);
        return s[it].l;
    }
    if (s[it].pushv > 0)
    {
        assert(s[it].pushv >= val);
        return s[it].l;
    }
    if (s[it].pushv == -1)
    {
        return max(val, s[it].l);
    }
    push(it);
    if (s[s[it].lc].max >= val)
        return binsearch_ge(s[it].lc, val);
    return binsearch_ge(s[it].rc, val);
}

ll pyramid_free_space(Pos45 v)
{
    // Domajam vairs tikai par konkreto piramidu
    int it = root[v.pos45 & 1];
    pint spos = v.pos45 / 2;
    pint epos = binsearch_ge(it, v.h45);
    if (epos <= spos)
        return 0;

    ll sum = getSum(it, spos, epos - 1);
    ll total_sum = (epos - spos) * (ll)v.h45;
    assert(sum <= total_sum);
    return total_sum - sum;
}


pair<pint, ll> binsearch_pyramid(pint pos, ll ballcnt)
{
    pint L = 1;
    pint R = MAXVS;

    pint hF = 0;
    ll sF = 0;

    while (L <= R)
    {
        pint h = (L + R) / 2;

        PosV v1 = {
            .h = h,
            .pos = pos
        };

        ll cur_ballcnt = pyramid_free_space(cpos45(v1));
        v1.h -= 1;
        cur_ballcnt += pyramid_free_space(cpos45(v1));

        if (cur_ballcnt <= ballcnt)
        {
            L = h + 1;
            hF = h;
            sF = cur_ballcnt;
        }
        else
        {
            R = h - 1;
        }
    }

    return {hF, sF};
}

void fill_pyramid(Pos45 v)
{
    // Domajam vairs tikai par konkreto piramidu
    int it = root[v.pos45 & 1];
    pint spos = v.pos45 / 2;
    pint epos = binsearch_ge(it, v.h45);
    if (epos <= spos)
        return;

    setSum(it, spos, epos - 1, v.h45);
}

pair<pint, ll> binsearch_and_fill_pyramid(pint pos, ll ballcnt)
{
    auto val = binsearch_pyramid(pos, ballcnt);
    PosV v1 = {
        .h = val.first,
        .pos = pos,
    };
    fill_pyramid(cpos45(v1));
    v1.h -= 1;
    fill_pyramid(cpos45(v1));
    return val;
}


int main()
{
    root[0] = buildTree(1, SIDEC * 2);
    root[1] = buildTree(1, SIDEC * 2);

    int q;
    scanf("%d", &q);
    for (; q > 0; q--)
    {
        pint pos;
        ll ballcnt;
        scanf("%d %lld", &pos, &ballcnt);
        pos = (-pos + SIDEC);

        pair<pint, ll> answ = binsearch_and_fill_pyramid(pos, ballcnt);

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
            pint h = answ.first + 1;
            assert(ballcnt > 0);

            PosV posv = {
                .h = h,
                .pos = pos
            };

            // Censamies izvietot atlikusas bumbas kreisaja mala

            auto pos45 = cpos45(posv);
            pint cnt45 = getSumPos(root[pos45.pos45 & 1], pos45.pos45 / 2);

            if (ballcnt > (pos45.h45 - 1 - cnt45))
            {
                // Aizpilda visu kreiso malu
                ballcnt -= (pos45.h45 - 1 - cnt45);
                setSum(root[pos45.pos45 & 1], pos45.pos45 / 2, pos45.pos45 / 2, pos45.h45 - 1);

                pint spos = pos45.pos45 / 2;
                pint epos = binsearch_ge(root[pos45.pos45 & 1], pos45.h45);
                assert(spos < epos);
                assert((epos - spos) >= ballcnt);
                pint tpos = epos - (pint)ballcnt;
                assert(spos <= tpos && tpos < epos);
                setSum(root[pos45.pos45 & 1], tpos, epos - 1, pos45.h45);
                answpos = cpos(Pos45 {
                    .h45 = pos45.h45,
                    .pos45 = tpos * 2 + (pos45.pos45 & 1),
                });
            }
            else
            {
                answpos = cpos({
                    .h45 = cnt45 + (pint)ballcnt,
                    .pos45 = pos45.pos45,
                });
                setSum(root[pos45.pos45 & 1], pos45.pos45 / 2, pos45.pos45 / 2, cnt45 + (int)ballcnt);
            }
        }

        answpos.pos = -(answpos.pos - SIDEC);
        printf("%d %d\n", answpos.pos, answpos.h);
    }
}
