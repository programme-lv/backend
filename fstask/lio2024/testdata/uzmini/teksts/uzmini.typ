#import "/template/template.typ": *

#let (
    conf,
    print_example, 
    print_example_raw,
    subtask_restriction_table, 
    restrictions_and_requirements,
    flush_description,
    contest,
    task,
) = prepare_task_document(
    contest_yaml: "/skola.yaml",
    task_codename: "Uzmini",
)

#show: doc => conf(
    doc,
)
#grid(
    columns: 2,
    gutter: 3pt,
    [ #align(bottom)[_Grūtība:_]], 
    [#image("/template/zvaigznes_4.png", height: 9pt);]
)

Dators "ir iedomājies" naturālu skaitli $X$ robežās no $1$ līdz $N$.

Skaitļa $X$ atminēšanai Jūsu programma var veikt _vaicājumus_. Katrs vaicājums ir formā "Vai iedomātais skaitlis ir $K$?", kur $1 <= K <= N$, un uz katru šādu vaicājumu dators dod vienu no trim atbildēm:
- $1$, ja $K < X$,
- $0$, ja $K = X$,
- $-1$, ja $K > X$.

Skaitlis $X$ ir uzminēts *tikai tad*, ja ir izdarīts vaicājums uz kuru saņemta atbilde $0$.

Katram vaicājumam ir noteikta _maksa_ -- ja vaicājumā izmatots skaitlis $K$, tad maksa par šādu vaicājumu ir $K$ eirocenti.

Uzrakstiet datorprogrammu, kas atrod skaitli $X$, iztērējot ne vairāk kā $3400$ eirocentus!

== Komunikācija

Šis ir interaktīvs uzdevums. Jūsu programmai, sākot darbu, pirmā ievada rinda
satur veselu skaitli $N$ ($1 <= N <= 500$). Iedomātā skaitļa vērtību $X$ vērtēšanas sistēma tur slepenībā. 
Tad jūsu programma var veikt vaicājumus, izvadā rakstot vērtību $K$ ($1 <= K <= N$). Vērtēšanas sistēma izdod atbildi nākamajā ievada rindā. Atbilde ir vesels skaitlis -- $-1, 0$ vai $1$, kā aprakstīts iepriekš. Jūsu programma katrā testā var veikt ne vairāk kā $N$ vaicājumus un nedrīkst iztērēt vairāk par $3400$ eirocentiem.
Kad uz vaicājumu tiek izdota atbilde $0$, jūsu programmai darbs jābeidz. 

== Piezīmes

//#flush_description()

 Lai nodrošinātu, ka jūsu vaicājumi tiek nodoti vērtēšanas sistēmai, jums ir
        jāsinhronizē (_flush_) izvada datu plūsma pēc katra vaicājuma:

       // #set text(size: 0.9em)
        #table(
            columns: 3,
            [_Valoda_], [_Piemērs_], [_Komentārs_],

            [C++],
            [
                ```txt
                std::cout << K << std::endl;
                ```
            ],
            [“`std::endl`” nodrošina datu plūsmas sinhronizāciju],

            [Go],
            [
                ```txt
                fmt.Println(K)
                ```
            ],
            [Standarta datu plūsma nav īpaši jāsinhronizē],

            [Java],
            [
                ```txt
                System.out.println(K);
                System.out.flush();
                ```
            ],
            [],

            [Pascal],
            [
                ```txt
                writeln(K);
                flush(output);
                ```
            ],
            [],

            [Python],
            [
                ```txt
                print(K, flush=True)
                ```
            ],
            [],
        )

Ja tiks pārsniegts maksimāli atļautais vaicājumu skaits, var tikt izdots kļūdas paziņojums “Izvaddati nav pareizi”. Šajā uzdevumā vērtēšanas sistēma darbojas adaptīvi -- tā pieskaņo savas atbildes lietotāja izvadam. Piemēram, vienam un tam pašam testam atbilde dažādām vaicājumu virknēm var atšķirties.
Izmantojot lietotāja testus sistēmas sadaļā “Testēšana”, ievaddatu faila vienīgajā rindā jānorāda $N$ vērtība.

#pagebreak()

== Piemērs

#table(
   columns: 8,
  align: horizon,
  [Ievaddati], [$6$], [],  [$1$], [], [$-1$], [],  [$0$],
  [Izvaddati (Jūsu programmas vaicājumi)], [], [$3$],  [], [$5$], [], [$4$],  [],
)

Pievērsiet uzmanību, ka, lai gan jau pēc otrā vaicājuma kļuva skaidrs, ka $X$ vērtība ir $4$, nācās veikt vēl trešo vaicājumu, lai saņemtu atbildi $0$. Šajā piemērā aprakstīto vaicājumu kopējās izmaksas bija $3 + 5 + 4 = 12$ eirocenti.


== Apakšuzdevumi un to vērtēšana

#subtask_restriction_table((
    none,
    [$N <= 5$],
    [$N <= 80$],
    [$N <= 400$],
    [Bez papildu ierobežojumiem],
))

