#include <bits/stdc++.h>

using namespace std;

using ll = long long;

const int MAXC = 1000000;
const int SIDEC = MAXC * 2;
const int MAXVS = MAXC; // Due to segment tree restrictions

int n;
int root[2];

const int MAXST = 2 * (2 * (SIDEC + 9));
struct STree {
    int l, r;
    int lc, rc;
    int pushv, max;
    ll total;
}s[MAXST];

int cntT = 0;

struct Pos45 {
    int h45;
    int pos45;
};

struct PosV {
    int h;
    int pos;
};

Pos45 cpos45(PosV val)
{
    int pos45 = val.pos - val.h;

    return Pos45 {
        .h45 = pos45 / 2 + val.h,
        .pos45 = pos45,
    };
}

PosV cpos(Pos45 val)
{
    int h = val.h45 - val.pos45 / 2;
    int pos = val.pos45 + h;

    return PosV {
        .h = h,
        .pos = pos,
    };
}

void update(int it)
{
    STree lc = s[s[it].lc];
    STree rc = s[s[it].rc];
    auto& cur = s[it];
    cur.total = lc.total + rc.total;
    cur.max = max(lc.max, rc.max);
}

void push_part(int cit, int pushv)
{
    auto& c = s[cit];
    c.max = max(c.max, pushv);
    c.total = (c.r - c.l + 1) * (ll)pushv;
    c.pushv = pushv;
}

void push(int it)
{
    if (s[it].pushv == 0)
        return;
    push_part(s[it].lc, s[it].pushv);
    push_part(s[it].rc, s[it].pushv);
    s[it].pushv = 0;
}

int buildTree(int l, int r, int odd)
{
    int it = ++cntT;
    assert(it < MAXST);
    s[it] = {
        .l = l,
        .r = r,
        .pushv = 0,
        .max = 0,
        .total = 0,
    };
    if (l == r) {
        s[it].lc = s[it].rc = 0;
        s[it].max = s[it].total = l;
    } else {
        int mid = l + (r - l) / 2;
        s[it].lc = buildTree(l, mid, odd);
        s[it].rc = buildTree(mid + 1, r, odd);
        update(it);
    }
    return it;
}

ll getSum(int it, int l, int r)
{
    if (l <= s[it].l && s[it].r <= r)
    {
        return s[it].total;
    }

    push(it);
    ll answ = 0;
    if (l <= s[s[it].lc].r)
        answ += getSum(s[it].lc, l, r);
    if (s[s[it].rc].l <= r)
        answ += getSum(s[it].rc, l, r);
    return answ;
}

int getSumPos(int it, int pos)
{
    return (int)getSum(it, pos, pos);
}

void setSum(int it, int l, int r, int val)
{
    if (l <= s[it].l && s[it].r <= r)
    {
        assert(s[it].max <= val);
        s[it].pushv = val;
        s[it].total = val * (ll)(s[it].r - s[it].l + 1);
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

int binsearch_ge(int it, int val)
{
    if (s[it].l == s[it].r)
    {
        assert(s[it].max >= val);
        return s[it].l;
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
    int spos = v.pos45 / 2;
    int epos = binsearch_ge(it, v.h45);
    if (epos <= spos)
        return 0;

    ll sum = getSum(it, spos, epos - 1);
    ll total_sum = (epos - spos) * (ll)v.h45;
    assert(sum <= total_sum);
    return total_sum - sum;
}


pair<int, ll> binsearch_pyramid(int pos, ll ballcnt)
{
    int L = 1;
    int R = MAXVS;

    int hF = 0;
    ll sF = 0;

    while (L <= R)
    {
        int h = (L + R) / 2;

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
    int spos = v.pos45 / 2;
    int epos = binsearch_ge(it, v.h45);
    if (epos <= spos)
        return;

    setSum(it, spos, epos - 1, v.h45);
}

pair<int, ll> binsearch_and_fill_pyramid(int pos, ll ballcnt)
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
    root[0] = buildTree(1, SIDEC, 0);
    root[1] = buildTree(1, SIDEC, 1);

    int q;
    scanf("%d", &q);
    for (; q > 0; q--)
    {
        int pos;
        ll ballcnt;
        scanf("%d %lld", &pos, &ballcnt);
        pos = (-pos + SIDEC);

        pair<int, ll> answ = binsearch_and_fill_pyramid(pos, ballcnt);

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
            int h = answ.first + 1;
            assert(ballcnt > 0);

            PosV posv = {
                .h = h,
                .pos = pos
            };

            // Censamies izvietot atlikusas bumbas kreisaja mala

            auto pos45 = cpos45(posv);
            int cnt45 = getSumPos(root[pos45.pos45 & 1], pos45.pos45 / 2);
            if (ballcnt > (pos45.h45 - 1 - cnt45))
            {
                // Aizpilda visu kreiso malu
                ballcnt -= (pos45.h45 - 1 - cnt45);
                setSum(root[pos45.pos45 & 1], pos45.pos45 / 2, pos45.pos45 / 2, pos45.h45 - 1);

                int spos = pos45.pos45 / 2;
                int epos = binsearch_ge(root[pos45.pos45 & 1], pos45.h45);
                assert(spos < epos);
                assert((epos - spos) >= ballcnt);
                int tpos = epos - (int)ballcnt;
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
                    .h45 = cnt45 + (int)ballcnt,
                    .pos45 = pos45.pos45,
                });
                setSum(root[pos45.pos45 & 1], pos45.pos45 / 2, pos45.pos45 / 2, cnt45 + (int)ballcnt);
            }
        }

        answpos.pos = -(answpos.pos - SIDEC);
        printf("%d %d\n", answpos.pos, answpos.h);
    }
}
