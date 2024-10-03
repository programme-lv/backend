#import "/template/template.typ": *

#let (
    conf,
    print_example, 
    print_example_raw,
    subtask_restriction_table, 
    restrictions_and_requirements,
    contest,
    task,
) = prepare_task_document(
    contest_yaml: "/skola.yaml",
    task_codename: "Kp",
)

#show: doc => conf(
    doc,
)
#grid(
    columns: 2,
    gutter: 3pt,
    [ #align(bottom)[_Grūtība:_]], 
    [#image("/template/zvaigznes_3.png", height: 9pt);]
)

Krišjānis ir uzkonstruējis kvadrātveida putekļsūcēju (saīsināti -- KP), kas ir neaizstājams palīgs viņa darbnīcas uzkopšanā. KP atmiņā darbnīcas grīda tiek attēlota kā $N times M$ rūtiņu laukums, kurā pats KP aizņem $K times K$ rūtiņas. Laukumā dažas rūtiņas var būt _bīstamas_ (netērēsim laiku, mēģinot noskaidrot, ko _tieši_ tas nozīmē), un KP nekad nedrīkst nonākt situācijā, ka KP atrašanās vietā vairāk nekā puse tā noklāto rūtiņu ir bīstamas. Ir zināma KP sākotnējā atrašanās vieta un _īpaša_ rūtiņa, kura _noteikti_ jāuzkopj, t.i., KP jāuzbrauc uz tās. Vienā solī KP var pārvietoties par vienu rūtiņu horizontālā vai vertikālā virzienā, neizejot no laukuma robežām. Nepieciešams noteikt, ar kādu mazāko soļu skaitu KP var nonākt situācijā, ka tas uzkopj īpašo rūtiņu.

Piemēram, @plans attēlā parādītajā kartē $N=5, M=9, K=3$ ar "A" apzīmēta KP sākotnējās atrašanās vietas kreisā augšējā rūtiņa, bet ar "B" -- īpašā rūtiņa. Bīstamās rūtiņas apzīmētas ar "X".

#figure(
    caption: [Laukuma piemērs],
    image("kp1.png", height: 8em)
)<plans>

Šajā gadījumā īpašo rūtiņu iespējams uzkopt ātrākais 10 soļos, veicot @marsruts attēlā parādīto maršrutu.

#figure(
    caption: [Īsākais maršruts],
    image("kp2.png", height: 28em)
)<marsruts>

Uzrakstiet datorprogrammu, kas dotam laukuma aprakstam nosaka, ar kādu mazāko soļu skaitu KP no sākuma pozīcijas var nonākt līdz īpašās rūtiņas apkopšanai!

== Ievaddati

Ievaddatu pirmajā rindā dotas trīs naturālu skaitļu -- laukuma rindu skaits~$N (2 <= N)$, laukuma kolonnu skaits~$M (2 <= M )$ un KP malas garums rūtiņās~$K (1 <= K <= min(N,M))$. 
Tiek garantēts, ka $N dot M <= 10^6$.
Starp katriem diviem blakus skaitļiem ievaddatos ir tukšumzīme.
Nākamajās $N$ ievaddatu rindās dots laukuma apraksts. Katrā rindā ir tieši $M$ simboli un katram $i (1 <= i <= N)$ un $j (1 <= j <= M)$ simbols ievaddatu $(i+1)$-ās rindas $j$-tajā kolonnā atbilst laukuma $i$-tās rindas $j$-tās kolonnas rūtiņas saturam un var būt:
- .(punkts) -- parasta rūtiņa
- X -- bīstama rūtiņa
- A -- KP sākuma atrašanās vietas kreisā augšējā stūra rūtiņa. Šī vienmēr ir parasta rūtiņa un uzdota korekti -- t.i., KP pilnībā ietilpst laukumā.
- B -- īpašā rūtiņa. Šī vienmēr ir parasta rūtiņa.  

== Izvaddati

Izvaddatu vienīgajā rindā jābūt veselam skaitlim -- mazākajam soļu skaitam, kas ļauj no sākuma pozīcijas nonākt līdz īpašās rūtiņas uzkopšanai, vai $-1$, ja derīgs maršruts neeksistē. Īpašā rūtiņa ir uzkopta tad, ja KP nonāk pozīcijā, kur tas noklāj īpašo rūtiņu. Ievērojiet, ka nekādi nav noteikts, kurai KP rūtiņai uzkopšanas brīdī jāatbilst īpašajai rūtiņai!

== Ierobežojumi un prasības

#restrictions_and_requirements()

// #pagebreak()

== Piemēri

#grid(
    columns: (60%, 40%),
    gutter: 1em,
    [
        #print_example(
            "kp.i00",
        )
    ],
)

== 1. apakšuzdevuma testu ievaddati

#grid(columns: 3, gutter: 1em, 
    [
        #print_example(
            "kp.i01a", 
            output: false,
        )
    ],
    [
        #print_example(
            "kp.i01b", 
            output: false,
        )
    ],
    [
        #print_example(
            "kp.i01c", 
            output: false,
        )
    ],
)

#block(breakable: false,
[
== Apakšuzdevumi un to vērtēšana

#subtask_restriction_table((
    none,
    [Uzdevuma tekstā dotie trīs piemēri],
    [$N M <= 10^3$],
    [$N <= 1000$ un $M <= 1000$],
    [Bez papildu ierobežojumiem],
))
]
)