# BORAIF — PROMPT-MESTRE DE ESPECIFICAÇÃO E IMPLEMENTAÇÃO

## 1. Missão

Você é o principal arquiteto e desenvolvedor responsável por projetar e implementar o sistema **BoraIF**.

O BoraIF é uma aplicação web para elaboração, organização, revisão, reutilização e posterior montagem de cadernos de questões voltados ao Ensino Médio brasileiro.

As disciplinas inicialmente suportadas são:

- Língua Portuguesa
- Língua Inglesa
- Física
- Química
- Redação
- História
- Geografia
- Matemática
- Biologia
- Artes
- Educação Física
- Filosofia
- Sociologia

O sistema deve ser simples, rápido, confiável, fácil de manter por muitos anos e deliberadamente evitar complexidade arquitetural desnecessária.

A funcionalidade central e mais importante do BoraIF é o **editor de questões**. Todas as demais funcionalidades devem ser projetadas em torno dele.

---

# 2. Princípios arquiteturais fundamentais

## 2.1 Monólito simples

O BoraIF deve ser um **monólito modular**, e não uma arquitetura de microservices.

NÃO implementar:

- microservices;
- Kubernetes;
- Redis inicialmente;
- RabbitMQ;
- Kafka;
- service mesh;
- API gateway separado;
- event bus;
- infraestrutura distribuída;
- múltiplos serviços independentes para funcionalidades que possam permanecer no backend principal.

Não introduzir infraestrutura ou abstrações complexas sem uma necessidade concreta e documentada.

O objetivo é maximizar:

- simplicidade;
- velocidade;
- confiabilidade;
- facilidade de manutenção;
- facilidade de deploy;
- facilidade de atualização;
- facilidade de entendimento do código.

## 2.2 Performance percebida

**O sistema deve ser projetado para proporcionar sensação de resposta imediata ao usuário. Operações CRUD comuns devem normalmente retornar em poucas centenas de milissegundos, não sendo aceitável bloquear a interface enquanto operações demoradas são executadas.**

Operações como:

- abrir uma questão;
- salvar uma questão;
- pesquisar;
- filtrar;
- carregar assuntos;
- carregar imagens;
- alterar metadados;
- listar questões;

devem ser rápidas.

Operações potencialmente demoradas, especialmente geração de PDF, devem ser executadas em background.

## 2.3 Simplicidade operacional

A aplicação será usada por poucas pessoas simultaneamente, aproximadamente 1–4 usuários simultâneos, embora ao longo dos anos possa haver muitos usuários cadastrados.

Não projetar para uma escala que não existe.

O sistema deve ser robusto, mas não superdimensionado.

---

# 3. Stack tecnológica obrigatória/preferencial

## Backend

Usar:

- Go
- HTTP/REST
- JSON
- PostgreSQL

O backend deve ser organizado de forma modular, clara e idiomática em Go, evitando frameworks excessivamente pesados quando bibliotecas simples forem suficientes.

## Frontend

Usar:

- React
- TypeScript

Não utilizar TanStack Query inicialmente.

O frontend deve ser cuidadosamente projetado para evitar chamadas HTTP redundantes.

## Editor

Usar **TipTap**, desde que seja possível utilizá-lo adequadamente com as licenças gratuitas/open-source necessárias ao projeto.

O editor é a parte central do sistema.

## Banco

PostgreSQL é obrigatório.

## Containers

Usar Docker Compose com exatamente três containers principais:

1. frontend;
2. backend;
3. PostgreSQL.

Chromium Headless para geração de PDF deverá funcionar dentro do container do backend, sem criar um quarto container inicialmente.

---

# 4. Regra crítica de persistência do banco

O PostgreSQL deve utilizar um volume Docker persistente.

É obrigatório garantir que operações normais como:

- rebuild do frontend;
- rebuild do backend;
- restart do frontend;
- restart do backend;
- recriação dos containers da aplicação;

NÃO apaguem nem recriem destrutivamente os dados do PostgreSQL.

O banco deve permanecer em um volume persistente independente do ciclo de vida dos containers frontend/backend.

O README deve explicar claramente:

- como iniciar o sistema;
- como fazer rebuild;
- como atualizar o backend;
- como atualizar o frontend;
- como reiniciar;
- como fazer backup do PostgreSQL;
- como restaurar o PostgreSQL;
- como atualizar a aplicação sem perder dados.

Nunca utilizar comandos destrutivos como comportamento normal de atualização.

---

# 5. Modelo conceitual de uma questão

Uma questão possui classificações/metadados:

- disciplina;
- assunto;
- ano do Ensino Médio;
- dificuldade;
- status;
- autor;
- data de criação;
- data de atualização;
- número simples de revisão.

## 5.1 Ano

Uma questão pertence a exatamente um ano:

- 1º ano;
- 2º ano;
- 3º ano.

Não permitir múltiplos anos para a mesma questão.

## 5.2 Dificuldade

Uma questão possui exatamente uma dificuldade:

- Fácil;
- Média;
- Difícil.

## 5.3 Status

Inicialmente suportar:

- RASCUNHO
- EM_REVISAO
- EM_TESTE
- TESTADA
- APROVADA
- REJEITADA
- ARQUIVADA
- OBSOLETA

Esses status devem ser configuráveis/representados adequadamente no banco, sem espalhar strings arbitrárias pelo código.

## 5.4 Revisão simples

NÃO implementar versionamento completo da questão neste momento.

A questão possui somente seu conteúdo atual.

Manter apenas um campo como:

`revision_number`

Esse número aumenta quando o conteúdo da questão sofre uma alteração relevante.

Exemplo:

Questão 152 — revisão 7.

Não armazenar as versões anteriores do conteúdo.

No futuro poderá existir versionamento completo, mas isso está fora do escopo atual.

---

# 6. Estrutura da questão

A questão deve ser estruturada, e NÃO armazenada como um único bloco indiferenciado de HTML.

Uma questão possui separadamente:

1. Enunciado
2. Comando
3. Alternativas

O enunciado deve ser um conteúdo independente.

O comando deve ser um conteúdo independente.

As alternativas também devem ser entidades/conteúdos independentes.

---

# 7. ALTERNATIVAS — REGRA FUNDAMENTAL

Esta regra é extremamente importante:

## Uma questão possui exatamente CINCO alternativas no sistema de prova.

As cinco alternativas são:

- A
- B
- C
- D
- E

NÃO existe uma sexta entidade chamada "correta".

A resposta correta é simplesmente uma propriedade de UMA das cinco alternativas existentes.

Exemplo:

- A — incorreta
- B — incorreta
- C — correta
- D — incorreta
- E — incorreta

O banco deve representar as alternativas de maneira separada da questão, permitindo tecnicamente que a estrutura seja extensível no futuro, mas a regra de negócio ATUAL deve permitir somente cinco alternativas utilizáveis por questão.

Portanto:

- uma questão deve possuir exatamente 5 alternativas válidas;
- as alternativas devem possuir uma ordem/posição;
- as posições atuais são A, B, C, D e E;
- exatamente uma delas deve ser marcada como correta;
- nenhuma questão pode ter zero alternativas;
- nenhuma questão pode ter mais de cinco alternativas;
- não pode haver duas alternativas corretas;
- não pode haver nenhuma alternativa correta.

É aceitável que a tabela de alternativas tenha uma estrutura que futuramente permita mais posições, mas o backend e a interface do BoraIF devem impor atualmente o limite de cinco.

A resposta correta deve ser identificada pela própria alternativa, por exemplo por um campo booleano `is_correct`, ou equivalente.

Não criar no modelo algo como:

`alternative_a`
`alternative_b`
`alternative_c`
`alternative_d`
`alternative_e`
`correct_answer`

se isso duplicar a informação.

Preferir uma tabela `question_alternatives` com cinco registros por questão, sendo exatamente um deles `is_correct = true`.

---

# 8. Conteúdo do TipTap

O conteúdo do:

- enunciado;
- comando;
- alternativa A;
- alternativa B;
- alternativa C;
- alternativa D;
- alternativa E;

deve ser armazenado em formato estruturado compatível com TipTap/ProseMirror JSON.

Não armazenar apenas HTML como representação principal.

O HTML poderá ser produzido quando necessário para:

- renderização;
- visualização;
- impressão;
- PDF.

Isso permitirá manter uma representação estruturada e previsível do conteúdo.

---

# 9. Editor de questões

O editor deve ser a funcionalidade mais refinada da aplicação.

O professor deve conseguir editar:

## Enunciado

Área TipTap independente.

## Comando

Área TipTap independente.

## Alternativas

Cinco áreas TipTap independentes:

- A
- B
- C
- D
- E

E deve haver uma maneira extremamente clara de indicar qual das cinco é a correta.

Exemplo conceitual:

A ○
B ○
C ●
D ○
E ○

A interface deve impedir ambiguidades.

---

# 10. Recursos do editor

O editor deve suportar, conforme as capacidades gratuitas/open-source escolhidas:

- texto normal;
- negrito;
- itálico;
- sublinhado;
- listas;
- alinhamento;
- links;
- imagens;
- tabelas, se necessário;
- subscrito;
- sobrescrito;
- fórmulas matemáticas;
- demais recursos que sejam úteis para questões escolares.

Não transformar o editor em um Word completo.

Adicionar apenas recursos que façam sentido para questões do Ensino Médio.

---

# 11. Fórmulas matemáticas

O sistema deve permitir DUAS formas de inserir fórmulas.

## 11.1 Entrada visual

O professor pode utilizar uma interface visual para construir a fórmula sem precisar conhecer LaTeX.

## 11.2 Entrada LaTeX

O professor também pode inserir diretamente código LaTeX.

Exemplo:

`\frac{-b \pm \sqrt{b^2-4ac}}{2a}`

O professor que conhece LaTeX poderá utilizá-lo.

O professor que não conhece LaTeX deverá conseguir utilizar a interface visual.

LaTeX deve ser utilizado principalmente como representação da fórmula, não como linguagem para construir o documento inteiro.

---

# 12. Renderização matemática

Avaliar e utilizar preferencialmente KaTeX, ou MathJax se houver razão técnica concreta para preferi-lo.

A arquitetura deve permitir que:

TipTap/representação da fórmula
→ HTML
→ renderização matemática
→ documento HTML final
→ Chromium Headless
→ PDF

Não gerar cada fórmula individualmente como um PDF.

---

# 13. Imagens

As imagens enviadas pelos professores serão compartilhadas entre todos os professores da mesma disciplina.

NÃO implementar autoria individual de imagens.

NÃO implementar permissões complexas para imagens.

A separação será principalmente por disciplina.

Exemplo:

`Física`
→ banco de imagens de Física.

Professores cadastrados em Física poderão reutilizar imagens desse banco.

O sistema deve permitir:

- upload;
- visualização;
- busca;
- seleção;
- reutilização;
- inserção no editor.

O caminho físico do arquivo pode ficar no banco como referência, enquanto o arquivo propriamente dito permanece no filesystem.

A solução inicial deve utilizar filesystem, não armazenamento em nuvem.

---

# 14. Assuntos

Cada questão pertence a exatamente UM assunto.

Cada assunto pertence a exatamente UMA disciplina.

O administrador pode cadastrar assuntos.

Porém, um professor também pode criar um novo assunto, desde que seja da disciplina à qual ele está cadastrado.

Exemplo:

Professor de Física:
→ pode criar "Mecânica dos Fluidos".

Não pode criar assunto de Matemática.

O sistema deve evitar duplicações acidentais de assuntos, realizando validação de nomes semelhantes/exatos antes de criar um novo assunto.

Os assuntos devem ser reutilizáveis e compartilhados entre os professores da mesma disciplina.

Não permitir que cada questão tenha um texto livre de assunto.

O assunto deve ser uma entidade própria no banco.

---

# 15. Disciplina do professor

Cada elaborador é associado a exatamente uma disciplina.

Por exemplo:

Professor João
→ Física.

Ele poderá elaborar questões de Física.

Inicialmente, um elaborador poderá visualizar e editar questões de outros elaboradores da mesma disciplina.

Deve existir uma configuração administrativa futura/preparada para permitir:

- elaborador vê todas as questões da disciplina;
ou
- elaborador vê somente suas próprias questões.

A configuração inicial deve permitir o comportamento atual: todos os elaboradores da mesma disciplina podem trabalhar com as questões daquela disciplina.

---

# 16. Assistente de IA

A IA NÃO é geradora automática de questões.

Ela é um **assistente de revisão e crítica**.

O professor continua sendo o autor da questão.

O editor deverá possuir ferramentas para solicitar análise de:

### Enunciado

A IA deve analisar, quando solicitado:

- clareza;
- precisão;
- ambiguidade;
- gramática;
- adequação ao ano;
- adequação pedagógica;
- possíveis problemas factuais;
- sugestões de melhoria.

### Comando

Analisar:

- clareza;
- coerência;
- objetividade;
- se realmente pergunta aquilo que as alternativas respondem.

### Alternativas/distratores

Analisar:

- plausibilidade;
- qualidade dos distratores;
- existência de pistas;
- ambiguidade;
- possibilidade de mais de uma resposta correta;
- qualidade da alternativa correta.

### Questão inteira

Analisar:

- coerência geral;
- relação entre enunciado, comando e alternativas;
- existência de exatamente uma alternativa correta;
- adequação ao ano;
- dificuldade;
- qualidade pedagógica;
- clareza;
- eventuais problemas.

A IA deve retornar críticas e sugestões.

O professor decide se aceita ou não.

Não substituir silenciosamente o conteúdo do professor.

---

# 17. API Key OpenAI

Cada professor deverá cadastrar sua própria API Key.

Não utilizar uma única API Key global para todos os professores.

A API Key deve ser armazenada criptografada no banco.

NUNCA armazenar a API Key em texto puro.

A chave de criptografia NÃO deve ser armazenada no PostgreSQL junto com o ciphertext.

Utilizar um segredo externo à base de dados, preferencialmente variável de ambiente/secret protegido pelo ambiente Docker/servidor.

O backend poderá descriptografar a chave quando precisar fazer uma chamada à OpenAI.

O professor não precisa receber a chave descriptografada para utilizar o sistema.

Não utilizar CPF ou outro dado pessoal como chave criptográfica.

Implementar criptografia autenticada adequada, preferencialmente AES-256-GCM ou equivalente seguro, com nonce/IV apropriado.

---

# 18. Autosave

Autosave é obrigatório.

O objetivo é impedir que o professor perca trabalho enquanto está elaborando uma questão.

Porém:

## JAMAIS salvar a cada caractere.

O autosave deve:

- utilizar debounce e/ou intervalo controlado;
- salvar a questão inteira;
- não salvar individualmente cada campo;
- não gerar uma requisição a cada tecla;
- indicar visualmente o estado.

Exemplo:

`Salvando...`

`Salvo`

`Erro ao salvar`

O autosave deve considerar o conjunto:

- metadados;
- enunciado;
- comando;
- cinco alternativas.

A implementação deve evitar condições de corrida entre autosaves.

---

# 19. React e chamadas HTTP

NÃO utilizar TanStack Query inicialmente.

O frontend deve ser escrito cuidadosamente para não fazer múltiplas chamadas idênticas ao backend.

Evitar:

- efeitos mal configurados;
- dependências incorretas de useEffect;
- chamadas disparadas por múltiplos componentes para o mesmo recurso;
- refetch desnecessário;
- chamadas causadas apenas por re-renderizações;
- chamadas duplicadas durante montagem/desmontagem;
- chamadas repetidas ao navegar entre componentes.

Dados já carregados devem ser mantidos adequadamente no estado/contexto local quando fizer sentido.

Se vários componentes precisam do mesmo dado, compartilhar o estado em vez de cada componente fazer sua própria chamada.

Considerar corretamente o comportamento de desenvolvimento do React Strict Mode.

O sistema deve ser projetado para que abrir uma única questão resulte em um conjunto pequeno e racional de requisições, e não em várias requisições idênticas.

---

# 20. Autenticação e perfis

Criar sistema de login.

Perfis iniciais:

## ADMIN

Acesso completo.

Pode:

- gerenciar usuários;
- gerenciar disciplinas;
- gerenciar assuntos;
- gerenciar configurações;
- gerenciar aplicações;
- gerar provas;
- visualizar questões;
- gerenciar demais recursos administrativos.

## ELABORADOR

É um professor associado a uma disciplina.

Pode:

- criar questões;
- editar questões permitidas;
- visualizar questões da disciplina;
- criar assuntos para sua disciplina;
- gerenciar/reutilizar imagens da disciplina;
- utilizar o assistente de IA;
- configurar sua própria API Key;
- pesquisar e filtrar questões.

## GESTOR/APLICAÇÃO

Perfil administrativo voltado à gestão das aplicações.

Pode:

- criar aplicações;
- configurar aplicações;
- selecionar parâmetros;
- gerar cadernos;
- visualizar PDFs gerados.

O backend deve validar as permissões.

Não é suficiente esconder botões no frontend.

---

# 21. Aplicações

"Aplicação" é uma entidade própria.

Exemplos:

- 2026/1
- 2026/2
- 2027/1

Uma aplicação representa uma campanha/temporada específica de aplicação das provas.

Uma aplicação possui:

- nome;
- descrição opcional;
- status;
- criador;
- datas relevantes;
- um ou mais cadernos de prova (ver seção 21.1);
- informações sobre a(s) prova(s) gerada(s).

## 21.1 Cadernos de prova

Uma aplicação pode gerar **mais de um caderno de prova**.

Exemplo:

- Caderno 1
- Caderno 2

O mais comum é existirem 2 cadernos, mas o número não é fixo — pode haver
apenas 1, ou também 3 ou mais.

Cada caderno é uma entidade própria, subordinada à aplicação, e possui:

- nome/identificação (ex.: "Caderno 1");
- configuração própria (seções 22 e 23): anos, total de questões,
  quantidade por disciplina, dificuldade por disciplina, quantidade por
  assunto;
- congelamento próprio (seção 27): congelar um caderno não afeta a
  configuração dos demais cadernos da mesma aplicação;
- numeração própria das questões: cada caderno é numerado
  sequencialmente de 1 até o total de questões daquele caderno, sem
  lacunas. Exemplo: um caderno com 80 questões é numerado de 1 a 80,
  independentemente da numeração de outro caderno da mesma aplicação;
- seu próprio PDF gerado (seção 28/30): a geração do documento é uma
  operação por caderno, não por aplicação. Uma aplicação com 2 cadernos
  gera 2 arquivos PDF distintos, cada um com seu próprio estado
  (PENDING/PROCESSING/COMPLETED/FAILED).

O snapshot de questões (seção 26) também é por caderno: cada caderno
preserva a representação das questões que utilizou, de forma independente
dos demais cadernos da mesma aplicação.

---

# 22. Configuração padrão

O sistema deve possuir uma configuração padrão sugerida.

Ela deve fornecer valores pré-preenchidos para facilitar a criação de uma nova aplicação.

O usuário NÃO deve ser obrigado a preencher tudo do zero.

Ao criar uma aplicação:

1. o sistema cria a aplicação;
2. copia a configuração padrão;
3. o usuário pode alterar os valores;
4. a configuração é salva para aquela aplicação.

A configuração padrão continua independente.

Alterar uma aplicação não deve alterar a configuração padrão.

---

# 23. Parâmetros da aplicação

A configuração deve permitir selecionar:

## Ano(s)

- 1º ano;
- 2º ano;
- 3º ano.

Pode haver mais de um ano selecionado.

## Total de questões

Informar o número total desejado.

## Quantidade por disciplina

Exemplo:

Português: 10
Matemática: 10
Física: 5
etc.

O sistema deve validar se a soma corresponde ao total desejado.

## Dificuldade por disciplina

Permitir definir quantidades de:

- Fácil;
- Média;
- Difícil.

Exemplo para Matemática:

Fácil: 3
Média: 5
Difícil: 2

## Assunto

A configuração também deve permitir, quando desejado, controlar quantidade por assunto dentro da disciplina.

O modelo deve ser flexível para permitir:

Disciplina
→ Assunto
→ Quantidade

sem tornar obrigatório informar assunto para todas as disciplinas.

---

# 24. Validação antes da geração

Antes de gerar a prova, o sistema deve verificar se existe quantidade suficiente de questões que satisfaçam simultaneamente os filtros.

Exemplo:

Solicitado:

Física
1º ano
Cinemática
Difícil
5 questões

Disponível:

3 questões.

O sistema deve impedir a geração e informar claramente:

Solicitadas: 5
Disponíveis: 3

Mostrar quais combinações de filtros não podem ser atendidas.

Não iniciar uma geração de PDF destinada a falhar.

---

# 25. Seleção das questões

Criar um componente/backend responsável pela seleção das questões.

Ele deve considerar:

- aplicação;
- anos selecionados;
- disciplina;
- assunto;
- dificuldade;
- status apropriado.

Definir claramente quais status podem participar da geração da prova.

A configuração deve ser documentada.

Por padrão, questões como:

- APROVADA;
- TESTADA;

podem ser elegíveis, enquanto:

- RASCUNHO;
- EM_REVISÃO;
- EM_TESTE;
- REJEITADA;
- ARQUIVADA;
- OBSOLETA;

não devem entrar automaticamente em provas finais.

Essa regra deve ser configurável ou claramente isolada para futura alteração.

---

# 26. Snapshot para aplicação

Embora não exista versionamento completo das questões, quando uma questão for utilizada em um caderno/prova gerada (seção 21.1), o sistema deve preservar a representação utilizada naquele PDF.

O objetivo é garantir que o PDF histórico de um caderno não mude caso a questão seja editada posteriormente.

Não implementar histórico completo da questão.

Implementar somente o snapshot necessário associado ao caderno. Como cada caderno tem seleção e numeração próprias, o snapshot é feito por caderno, não por aplicação como um todo.

A questão atual continua sendo a única versão editável no banco principal.

---

# 27. Congelamento da configuração

Quando a prova de um caderno for efetivamente gerada, os parâmetros utilizados por aquele caderno devem ser congelados.

Depois disso, não permitir alteração dos parâmetros que definiram aquele caderno específico. Congelar um caderno não afeta a configuração dos demais cadernos da mesma aplicação (seção 21.1).

Se houver necessidade de uma nova configuração, deve existir uma nova geração/caderno ou outro mecanismo explicitamente definido, sem alterar silenciosamente a configuração histórica.

O PDF gerado deve permanecer associado ao caderno (e, por meio dele, à aplicação).

---

# 28. Geração de PDF

A solução preferencial é:

**HTML + CSS → Chromium Headless → PDF**

Não usar LaTeX como linguagem do documento completo.

Fluxo:

Questões estruturadas
→ HTML
→ CSS de impressão
→ renderização matemática
→ Chromium Headless
→ PDF.

O PDF deve ser adequado para impressão em papel A4.

Utilizar CSS de impressão.

Prever:

- margens;
- quebra de páginas;
- cabeçalho;
- rodapé;
- numeração;
- identificação da aplicação;
- número das questões;
- alternativas A–E;
- imagens;
- fórmulas.

Evitar quebras ruins de questão entre páginas quando tecnicamente possível.

---

# 29. Chromium Headless

Chromium Headless deve ser executado dentro do container backend.

O backend Go deverá controlar/acionar o Chromium.

Não criar um serviço separado apenas para PDF neste momento.

A geração pode utilizar uma biblioteca Go adequada, como chromedp, ou solução equivalente bem mantida.

Avaliar compatibilidade de:

- Linux;
- Docker;
- Chromium;
- CSS de impressão;
- KaTeX/MathJax;
- imagens locais;
- fontes;
- A4.

---

# 30. Geração em background

Gerar PDF pode demorar.

Essa operação NÃO deve bloquear a requisição HTTP principal.

A geração de PDF é uma ação por caderno (seção 21.1): uma aplicação com
múltiplos cadernos gera um PDF de cada vez que "Gerar PDF" é acionado para
um caderno específico, não um único PDF para toda a aplicação.

Fluxo:

Usuário clica, para um caderno específico:

`Gerar PDF`

Backend:

1. valida configuração;
2. valida disponibilidade das questões;
3. cria registro de geração;
4. inicia processo em background;
5. responde rapidamente.

Background:

1. seleciona questões;
2. cria snapshot;
3. monta HTML;
4. renderiza matemática;
5. executa Chromium;
6. gera PDF;
7. salva arquivo;
8. atualiza status.

Estados possíveis:

- PENDING
- PROCESSING
- COMPLETED
- FAILED

O frontend pode consultar o status sem realizar polling agressivo.

Não implementar sistema sofisticado de filas.

Um mecanismo simples de background dentro do próprio backend é suficiente.

---

# 31. Armazenamento de arquivos

Inicialmente utilizar filesystem.

Estruturar diretórios de forma organizada, por exemplo:

`uploads/`

`uploads/images/`

`uploads/images/{discipline}/`

`generated/`

`generated/applications/`

Os caminhos devem ser registrados no banco.

Não armazenar grandes arquivos binários diretamente no PostgreSQL neste primeiro momento.

Criar abstração simples de storage para permitir futura migração para S3 ou equivalente, se algum dia necessário, sem implementar S3 agora.

---

# 32. Banco de dados — obrigatório

O agente DEVE criar todo o banco de dados.

Não entregar apenas modelos/entidades da aplicação.

É obrigatório criar:

1. script completo de criação/migrations das tabelas;
2. constraints;
3. índices;
4. foreign keys;
5. unique constraints;
6. dados iniciais/seed;
7. inserts iniciais para tabelas que necessitem de valores padrão;
8. mecanismo de migration versionado.

O banco deve poder ser criado de forma reproduzível em uma instalação nova.

## Dados iniciais

Inserir inicialmente pelo menos:

### Disciplinas

- Língua Portuguesa
- Língua Inglesa
- Física
- Química
- Redação
- História
- Geografia
- Matemática
- Biologia
- Artes
- Educação Física
- Filosofia
- Sociologia

### Anos

- 1º ano
- 2º ano
- 3º ano

### Dificuldades

- Fácil
- Média
- Difícil

### Status

- Rascunho
- Em revisão
- Em teste
- Testada
- Aprovada
- Rejeitada
- Arquivada
- Obsoleta

Os nomes técnicos internos podem utilizar identificadores estáveis.

Criar também dados de exemplo quando isso ajudar no desenvolvimento e nos testes, deixando claramente identificados como dados de demonstração/seed.

O README deve explicar como criar o banco do zero.

---

# 33. Modelo de dados esperado

Projetar um modelo relacional normalizado.

Entidades esperadas incluem, mas não necessariamente limitadas a:

- users;
- roles, se necessário;
- disciplines;
- subjects;
- questions;
- question_alternatives;
- images;
- applications;
- application_booklets (cadernos de prova de uma aplicação — seção 21.1; uma aplicação pode ter mais de um caderno, cada um com configuração, congelamento, numeração e PDF próprios);
- booklet_configurations / booklet_quota_rules (configuração e cotas de seleção de cada caderno);
- booklet_question_snapshots (snapshot de questões por caderno, com a numeração 1..N daquele caderno);
- generated_documents (um registro de geração de PDF por caderno);
- configurações relevantes;
- registros de background jobs, se necessários.

O modelo final deve ser justificado.

## Regra das alternativas

`question_alternatives` deve possuir exatamente cinco alternativas por questão em estado válido.

Deve existir constraint/regra no backend garantindo:

- A;
- B;
- C;
- D;
- E;
- exatamente uma correta.

Avaliar também constraints PostgreSQL quando forem adequadas.

---

# 34. API REST

Projetar API REST clara.

Exemplos conceituais:

`POST /api/auth/login`

`GET /api/questions`

`POST /api/questions`

`GET /api/questions/{id}`

`PUT /api/questions/{id}`

`DELETE /api/questions/{id}`

`POST /api/questions/{id}/ai/review`

`GET /api/subjects`

`POST /api/subjects`

`POST /api/images`

`GET /api/images`

`GET /api/applications`

`POST /api/applications`

`GET /api/applications/{id}`

`PUT /api/applications/{id}/configuration`

`POST /api/applications/{id}/generate`

`GET /api/applications/{id}/generation-status`

Esses endpoints são exemplos e devem ser refinados durante o projeto.

Não criar endpoints excessivamente fragmentados sem necessidade.

---

# 35. Segurança

Implementar:

- senhas armazenadas como hash seguro;
- autenticação;
- autorização por perfil;
- autorização por disciplina;
- validação server-side;
- proteção de uploads;
- validação de MIME type;
- limite de tamanho de arquivos;
- nomes de arquivos seguros;
- proteção contra path traversal;
- sanitização/controle do HTML renderizado;
- proteção adequada das credenciais da OpenAI.

Nunca confiar apenas no frontend para segurança.

---

# 36. Interface principal

Criar uma interface limpa, profissional e rápida.

Principais áreas:

## Dashboard

Resumo relevante para o perfil.

## Questões

Listagem com:

- busca;
- disciplina;
- assunto;
- ano;
- dificuldade;
- status;
- autor, quando apropriado.

Para elaborador, a disciplina deve ser automaticamente limitada à sua disciplina.

## Nova questão

Editor completo.

## Editar questão

Mesmo editor.

## Imagens

Biblioteca de imagens da disciplina.

## Aplicações

Listagem e gestão das aplicações.

## Geração de prova

Tela de configuração e geração.

## PDFs

Histórico dos PDFs gerados.

## Administração

Usuários, disciplinas, assuntos, configurações etc.

---

# 37. UX do editor

O professor deve conseguir começar a escrever imediatamente.

Evitar telas excessivamente burocráticas.

Idealmente:

1. cria questão;
2. escolhe ano;
3. escolhe assunto;
4. escolhe dificuldade;
5. começa a escrever.

A disciplina já deve vir determinada pelo usuário.

O status inicial deve ser RASCUNHO.

Autosave deve começar automaticamente.

---

# 38. Listagem de questões

A listagem deve ser rápida e permitir:

- pesquisa por texto;
- assunto;
- ano;
- dificuldade;
- status;
- autor, quando aplicável.

Permitir ordenar por:

- atualização;
- criação;
- assunto;
- status;
- etc.

Utilizar paginação no backend.

Não carregar milhares de questões de uma vez.

---

# 39. Performance do banco

Criar índices adequados para os filtros mais utilizados.

Especialmente considerar índices envolvendo:

- discipline_id;
- subject_id;
- grade/year;
- difficulty;
- status;
- updated_at;
- author_id.

Não criar índices indiscriminadamente.

Medir e justificar os índices importantes.

---

# 40. Estrutura do projeto

A estrutura deve ser clara.

Exemplo conceitual:

```text
boraif/
├── backend/
├── frontend/
├── database/
├── uploads/
├── generated/
├── docker-compose.yml
├── .env.example
└── README.md
```

O backend Go deve possuir organização modular por domínio, sem exagero arquitetural.

Exemplo:

```text
backend/
├── cmd/
├── internal/
│   ├── auth/
│   ├── users/
│   ├── disciplines/
│   ├── subjects/
│   ├── questions/
│   ├── images/
│   ├── applications/
│   ├── ai/
│   ├── pdf/
│   └── storage/
├── migrations/
├── templates/
├── go.mod
└── ...
```

Adaptar se houver uma estrutura Go melhor justificada.

---

# 41. Docker

Criar:

- Dockerfile do frontend;
- Dockerfile do backend;
- configuração PostgreSQL;
- docker-compose.yml.

O frontend deve ser servido de maneira adequada para produção.

O backend deve ser compilado de maneira eficiente.

A imagem final do backend deve conter Chromium Headless e suas dependências necessárias.

Garantir funcionamento em Ubuntu/Docker.

---

# 42. Configuração

Nunca colocar secrets diretamente no código.

Utilizar `.env` apenas para configuração local, e `.env.example` sem secrets reais.

Configurações esperadas:

- PostgreSQL;
- porta backend;
- porta frontend;
- segredo de autenticação;
- chave de criptografia das API Keys;
- configurações do storage;
- configurações do Chromium;
- outras configurações necessárias.

---

# 43. Testes

Criar testes para as regras de negócio críticas.

Especialmente:

## Questões

- criação;
- edição;
- exatamente cinco alternativas;
- exatamente uma correta;
- disciplina;
- assunto;
- ano;
- dificuldade;
- status.

## Permissões

- admin;
- elaborador;
- gestor;
- disciplina.

## Autosave

Testar comportamento do endpoint de atualização.

## Aplicações

- configuração;
- validação;
- congelamento;
- seleção de questões.

## Seleção

Testar combinações de:

- disciplina;
- assunto;
- ano;
- dificuldade.

## PDF

Criar pelo menos um teste de geração/renderização que verifique que um documento válido é produzido.

---

# 44. Dados de demonstração

Criar seed/demo data suficiente para demonstrar o sistema.

Incluir algumas questões fictícias para diferentes disciplinas, anos, dificuldades e assuntos.

As questões de demonstração devem deixar claro que são dados de teste.

Criar imagens de demonstração somente se necessário e sem utilizar material protegido sem permissão.

---

# 45. Critérios de aceitação principais

O BoraIF somente deve ser considerado funcional quando:

1. usuário consegue fazer login;
2. permissões funcionam;
3. elaborador está associado a uma disciplina;
4. elaborador consegue criar uma questão;
5. questão possui enunciado separado;
6. questão possui comando separado;
7. questão possui exatamente cinco alternativas A–E;
8. exatamente uma das cinco alternativas pode ser correta;
9. questão possui assunto;
10. questão possui ano;
11. questão possui dificuldade;
12. questão possui status;
13. questão salva automaticamente sem salvar a cada caractere;
14. questão pode ser salva manualmente;
15. questões podem ser pesquisadas/filtradas;
16. imagens podem ser carregadas e reutilizadas pela disciplina;
17. fórmulas podem ser inseridas visualmente;
18. fórmulas podem ser inseridas usando LaTeX;
19. professor pode solicitar análise da IA;
20. API Key individual é armazenada criptografada;
21. aplicações podem ser criadas;
22. configuração padrão é copiada para nova aplicação;
23. configuração pode ser modificada antes da geração;
24. sistema verifica disponibilidade antes de gerar;
25. configuração usada para a prova é congelada;
26. prova é gerada em background;
27. Chromium Headless gera PDF;
28. PDF é armazenado;
29. PDF permanece associado à aplicação;
30. Docker Compose sobe frontend/backend/PostgreSQL;
31. rebuild do backend não destrói PostgreSQL;
32. CRUD comum responde rapidamente;
33. frontend não realiza chamadas HTTP redundantes desnecessárias.

---

# 46. Estratégia de implementação

NÃO tente implementar todo o sistema de uma só vez.

Trabalhe por fases.

## Fase 1 — Fundação

- estrutura Git;
- Docker;
- PostgreSQL;
- migrations;
- seed;
- backend Go;
- frontend React;
- autenticação básica.

Ao final, sistema deve subir com os três containers.

## Fase 2 — Usuários e permissões

- usuários;
- perfis;
- disciplina;
- autorização.

## Fase 3 — Assuntos

- CRUD;
- associação à disciplina;
- criação por administrador;
- criação pelo elaborador dentro da própria disciplina.

## Fase 4 — Questões

Implementar primeiro o CRUD estrutural.

## Fase 5 — Editor TipTap

Implementar:

- enunciado;
- comando;
- A–E;
- imagens;
- formatação;
- fórmulas.

## Fase 6 — Autosave

Implementar e testar cuidadosamente.

## Fase 7 — Biblioteca de imagens

Upload, armazenamento, busca e reutilização por disciplina.

## Fase 8 — Assistente OpenAI

Implementar revisão de:

- enunciado;
- comando;
- alternativas;
- questão completa.

## Fase 9 — Aplicações

Implementar:

- entidade;
- configuração;
- configuração padrão;
- filtros;
- seleção.

## Fase 10 — Geração PDF

Implementar:

- snapshot;
- HTML;
- CSS;
- KaTeX/MathJax;
- Chromium;
- background job;
- armazenamento.

## Fase 11 — Refinamento

- performance;
- UX;
- testes;
- segurança;
- documentação;
- backup;
- deploy.

Ao final de cada fase, garantir que o sistema continue compilando, iniciando e funcionando.

---

# 47. Regra para decisões técnicas

Quando houver duas soluções possíveis, preferir a que:

1. possui menos dependências;
2. possui menos infraestrutura;
3. é mais fácil de entender;
4. é mais fácil de atualizar;
5. é mais fácil de testar;
6. reduz chamadas de rede;
7. reduz estados distribuídos;
8. reduz pontos de falha.

Não escolher uma tecnologia apenas porque é popular.

---

# 48. O que NÃO fazer

Não:

- transformar o sistema em microservices;
- adicionar Kubernetes;
- adicionar Redis sem necessidade;
- adicionar RabbitMQ;
- adicionar Kafka;
- criar serviço separado para PDF;
- criar serviço separado para IA;
- criar uma arquitetura distribuída;
- armazenar API Keys em texto puro;
- utilizar CPF como chave criptográfica;
- salvar questão a cada caractere;
- fazer uma requisição HTTP para cada campo;
- fazer múltiplas chamadas idênticas desnecessárias;
- criar seis alternativas;
- criar uma entidade separada chamada "resposta correta";
- armazenar alternativas A–E como cinco colunas independentes se uma tabela relacional for mais adequada;
- implementar versionamento completo das questões;
- criar autorização complexa para imagens;
- criar armazenamento cloud de imagens inicialmente;
- carregar todas as questões da base de uma vez;
- bloquear a interface durante geração de PDF;
- apagar PostgreSQL durante rebuild;
- adicionar funcionalidades não solicitadas apenas para tornar a arquitetura "mais completa".

---

# 49. Importante: não inventar requisitos

Quando encontrar uma decisão não especificada:

1. verificar se existe uma decisão implícita nesta especificação;
2. escolher a solução mais simples e consistente;
3. documentar a decisão;
4. não criar funcionalidades grandes sem necessidade.

Se uma decisão puder alterar significativamente a arquitetura, parar na fase de projeto e apresentar a decisão antes de implementar.

---

# 50. Entregáveis esperados

O projeto deve entregar:

- código completo do frontend;
- código completo do backend;
- Dockerfiles;
- docker-compose.yml;
- migrations PostgreSQL;
- scripts de criação;
- seeds/inserts iniciais;
- documentação;
- `.env.example`;
- testes;
- README completo;
- instruções de desenvolvimento;
- instruções de produção;
- instruções de backup/restauração;
- documentação básica da API;
- documentação das principais decisões arquiteturais.

O banco de dados deve poder ser criado novamente do zero em outro ambiente.

---

# 51. README obrigatório

O README deve explicar, de maneira prática:

1. pré-requisitos;
2. como clonar;
3. como configurar `.env`;
4. como subir o Docker;
5. como criar banco;
6. como executar migrations;
7. como executar seeds;
8. como acessar frontend;
9. como acessar backend;
10. como criar primeiro administrador;
11. como cadastrar disciplinas;
12. como cadastrar usuários;
13. como cadastrar assuntos;
14. como cadastrar API Key;
15. como fazer rebuild;
16. como atualizar;
17. como fazer backup;
18. como restaurar;
19. onde ficam imagens;
20. onde ficam PDFs.

Incluir uma seção explícita:

**"Como fazer rebuild sem perder o banco de dados"**

---

# 52. Primeira tarefa do agente

Antes de escrever grande quantidade de código:

1. analisar toda esta especificação;
2. identificar eventuais ambiguidades;
3. apresentar arquitetura proposta;
4. apresentar modelo relacional PostgreSQL;
5. apresentar estrutura dos três containers;
6. apresentar estratégia HTML → Chromium → PDF;
7. apresentar estratégia TipTap → JSON → HTML;
8. apresentar estratégia KaTeX/MathJax;
9. apresentar estratégia de autosave;
10. apresentar estratégia de chamadas HTTP do React;
11. apresentar estratégia de criptografia das API Keys;
12. apresentar fases de implementação.

Depois disso, iniciar a implementação de forma incremental.

Não gerar código enorme sem validação intermediária.

---

# 53. Filosofia geral do BoraIF

O BoraIF deve parecer uma aplicação simples para o usuário, mesmo que internamente possua uma modelagem bem organizada.

A experiência ideal é:

**Professor entra → cria questão → começa a escrever → sistema salva automaticamente → professor revisa com IA → salva → questão fica disponível no banco → posteriormente uma aplicação utiliza as questões → PDF é gerado em background.**

O sistema deve evitar burocracia.

A questão é o coração do produto.

A qualidade do editor, a velocidade de edição, o autosave, a organização das questões e a capacidade de reutilização são prioridades superiores a funcionalidades administrativas sofisticadas.

**Construir menos, mas construir muito bem.**
