# generic
generic type inference implementaion for go-like grammar language

## 형식적 정의

### 타입 환경 (Γ)

타입 환경 Γ는 식별자와 타입의 매핑을 나타냅니다:

$\Gamma : \text{Identifier} \rightarrow \text{Type}$

예: $\Gamma = \{x : \text{int}, f : \text{int} \rightarrow \text{string}\}$

### 타입 추론 (InferType)

타입 추론은 주어진 표현식(Expr)과 타입 환경(Env)을 기반으로 해당 표현식의 타입(Type)을 결정하는 과정입니다. 타입 환경($\Gamma$)은 식별자와 그 타입을 매핑하는 역할을 합니다. 타입 추론 과정을 통해 표현식의 타입을 유도하여 프로그램이 타입 안전성을 유지할 수 있도록 합니다.

타입 추론 과정은 다음과 같습니다:

$\text{InferType} : \text{Expr} \times \text{Env} \rightarrow \text{Type}$

$$
\text{InferType}(e, \Gamma) = \begin{cases}
    \tau & \text{if}\ e : \text{Literal} \\
    \Gamma(x) & \text{if}\ e = x : \text{Identifier} \\
    \tau_2 & \text{if}\ e = e_1\ e_2 \land \text{InferType}(e_1, \Gamma) = \tau_1 \rightarrow \tau_2 \land \text{InferType}(e_2, \Gamma) = \tau_1 \\
    \tau_1 \rightarrow \tau_2 & \text{if}\ e = \lambda x:\tau_1. e' \land \text{InferType}(e', \Gamma[x \mapsto \tau_1]) = \tau_2 \\
    \forall \alpha. \tau & \text{if}\ e = \Lambda \alpha. e' \land \text{InferType}(e', \Gamma[\alpha \mapsto \text{TypeVar}]) = \tau \\
    \text{SliceType}(\tau) & \text{if}\ e = [\tau] \land \forall e_i \in e, \text{InferType}(e_i, \Gamma) = \tau \\
    \text{MapType}(\tau_1, \tau_2) & \text{if}\ e = \text{map}[\tau_1]\tau_2 \land \forall (k_i, v_i) \in e, \text{InferType}(k_i, \Gamma) = \tau_1 \land \text{InferType}(v_i, \Gamma) = \tau_2
\end{cases}
$$

1. **리터럴 (Literal)**:

- 리터럴 값(e.g., 숫자, 문자열 등)의 타입은 해당 리터럴의 타입($\tau$)입니다.
- 예: `InferType(42, Γ) = int`

2. **식별자 (Identifier)**:

- 식별자의 타입은 타입 환경($\Gamma$)에서 식별자에 매핑된 타입입니다.
- 예: `InferType(x, Γ[x ↦ int]) = int`

3. **함수 적용 (Function Application)**:

- 표현식 $e_1$과 $e_2$의 타입을 추론하여 $e_1$의 타입이 함수 타입($\tau_1 \rightarrow \tau_2$)이고 $e_2$의 타입이 $\tau_1$일 때, 전체 표현식 $e = e_1\ e_2$의 타입은 $\tau_2$입니다.
- 예: `InferType(f(x), Γ[f ↦ (int → string), x ↦ int]) = string`

4. **함수 정의 (Function Definition)**:

- 람다 표현식($\lambda x:\tau_1. e'$)의 타입을 추론할 때, 인자 $x$의 타입을 $\tau_1$로 가정하고 본문 $e'$의 타입을 추론합니다. 이 결과 $\tau_2$가 나오면 전체 표현식의 타입은 $\tau_1 \rightarrow \tau_2$입니다.
- 예: `InferType(\lambda x:int. x + 1, Γ) = int → int`

5. **제네릭 함수 (Generic Function)**:

- 제네릭 함수 표현식($\Lambda \alpha. e'$)의 타입을 추론할 때, 타입 변수 $\alpha$를 새로운 타입 변수로 가정하고 본문 $e'$의 타입을 추론합니다. 이 결과 $\tau$가 나오면 전체 표현식의 타입은 $\forall \alpha. \tau$입니다.
- 예: `InferType(Λα. λx:α. x, Γ) = ∀α. α → α`

6. **슬라이스 (Slice)**:

- 슬라이스 표현식(e.g., 배열)의 모든 요소 $e_i$가 동일한 타입 $\tau$를 가질 때, 슬라이스의 타입은 $\text{SliceType}(\tau)$입니다.
- 예: `InferType([1, 2, 3], Γ) = SliceType(int)`

7. **맵 (Map)**:

- 맵 표현식의 모든 키 $k_i$가 타입 $\tau_1$를 가지고, 모든 값 $v_i$가 타입 $\tau_2$를 가질 때, 맵의 타입은 $\text{MapType}(\tau_1, \tau_2)$입니다.
- 예: `InferType(map[string]int{"a": 1, "b": 2}, Γ) = MapType(string, int)`

### 타입 통합 (Unify)

**자유 타입 변수 (Free Type Variables)**

자유 타입 변수(FTV)는 어떤 타입 표현식 내에서 바인딩되지 않은 타입 변수를 의미합니다. 예를 들어, 타입 $\alpha \rightarrow \beta$에서 $\alpha$와 $\beta$는 자유 타입 변수입니다. 이들은 타입 시스템 내에서 다른 타입으로 대체될 수 있습니다.

**타입 통합 (Type Unification)**

타입 통합은 두 타입을 비교하여 이들이 동일한지, 혹은 하나의 타입 변수가 다른 타입으로 대체될 수 있는지를 확인하는 과정입니다. 타입 통합 과정에서 타입 환경($\Gamma$)이 갱신되며, 이는 식별자와 타입 변수의 매핑을 포함합니다. 타입 통합의 결과는 갱신된 타입 환경입니다.

타입 통합은 다음과 같이 정의됩니다:

$\text{Unify} : \text{Type} \times \text{Type} \times \text{Env} \rightarrow \text{Env}$

$$
\text{Unify}(\tau_1, \tau_2, \Gamma) = \begin{cases}
    \Gamma & \text{if}\ \tau_1 = \tau_2 \\
    \Gamma[\alpha \mapsto \tau_2] & \text{if}\ \tau_1 = \alpha \land \alpha \notin \text{FTV}(\tau_2) \\
    \Gamma[\alpha \mapsto \tau_1] & \text{if}\ \tau_2 = \alpha \land \alpha \notin \text{FTV}(\tau_1) \\
    \text{Unify}(\tau_1', \tau_2', \text{Unify}(\tau_1'', \tau_2'', \Gamma)) & \text{if}\ \tau_1 = \tau_1' \rightarrow \tau_1'' \land \tau_2 = \tau_2' \rightarrow \tau_2'' \\
    \text{Unify}(\tau', \tau'', \Gamma) & \text{if}\ \tau_1 = \text{SliceType}(\tau') \land \tau_2 = \text{SliceType}(\tau'') \\
    \text{Unify}(\tau_1', \tau_2', \text{Unify}(\tau_1'', \tau_2'', \Gamma)) & \text{if}\ \tau_1 = \text{MapType}(\tau_1', \tau_1'') \land \tau_2 = \text{MapType}(\tau_2', \tau_2'') \\
    \text{error} & \text{otherwise}
\end{cases}
$$

여기서 `FTV`는 자유 타입 변수(Free Type Variables)의 집합을 나타냅니다.

예시:

1. 동일 타입
    - `Unify(int, int, Γ) = Γ`
    - 동일한 두 타입을 비교하면 타입 환경은 변경되지 않습니다.

2. 타입 변수 바인딩
    - `Unify(α, int, Γ) = Γ[α ↦ int]`
    - 타입 변수가 다른 타입으로 바인딩될 수 있으면, 타입 환경을 갱신하여 해당 변수를 새로운 타입으로 매핑합니다.

3. 함수 타입
    - `Unify((int → α), (int → string), Γ) = Γ[α ↦ string]`
    - 함수 타입을 비교할 때, 각 인자와 반환값의 타입을 비교합니다.

4. 슬라이스 타입
    - `Unify(SliceType(α), SliceType(int), Γ) = Γ[α ↦ int]`
    - 슬라이스 타입의 경우, 요소 타입을 비교하여 통합합니다.

5. 맵 타입
    - `Unify(MapType(string, α), MapType(string, int), Γ) = Γ[α ↦ int]`
    - 맵 타입의 경우, 키와 값의 타입을 개별적으로 통합합니다.

복잡한 예:
> `Unify((α → β) → γ, (int → string) → bool, Γ)`
> ` = Unify(α → β, int → string, Unify(γ, bool, Γ))`
> ` = Unify(α, int, Unify(β, string, Γ[γ ↦ bool]))`
> ` = Γ[α ↦ int, β ↦ string, γ ↦ bool]`

### 제약 조건 검사 (checkConstraint)

제약 조건 검사는 특정 타입이 주어진 제약 조건을 만족하는지 확인하는 과정입니다. 제약 조건은 인터페이스 구현 여부와 특정 타입 집합에 속하는지를 포함할 수 있습니다. 제약 조건 검사는 타입 시스템에서 타입 안전성을 유지하고, 인터페이스를 통한 다형성을 보장하는 데 중요합니다.

$\text{checkConstraint} : \text{Type} \times \text{Constraint} \rightarrow \text{Bool}$

제약 조건은 다음과 같이 정의됩니다.

$$
\text{checkConstraint}(\tau, C) = \begin{cases}
 \bigwedge_{i \in C.\text{Interfaces}} \text{implements}(\tau, i) & \text{if}\ C.\text{Types} = \emptyset \\
 \bigvee_{t \in C.\text{Types}} \tau = t & \text{if}\ C.\text{Interfaces} = \emptyset \\
 (\bigwedge_{i \in C.\text{Interfaces}} \text{implements}(\tau, i)) \land (\bigvee_{t \in C.\text{Types}} \tau = t) & \text{otherwise}
\end{cases}
$$

이 정의는 다음과 같은 세 가지 경우를 다룹니다:

1. **인터페이스 제약 조건**: $C.\text{Types}$가 비어 있는 경우, 타입 $\tau$는 $C.\text{Interfaces}$의 모든 인터페이스를 구현해야 합니다.
2. **타입 제약 조건**: $C.\text{Interfaces}$가 비어 있는 경우, 타입 $\tau$는 $C.\text{Types}$에 포함된 타입 중 하나와 일치해야 합니다.
3. **복합 제약 조건**: 두 제약 조건이 모두 있는 경우, 타입 $\tau$는 $C.\text{Interfaces}$의 모든 인터페이스를 구현하고, 동시에 $C.\text{Types}$에 포함된 타입 중 하나와 일치해야 합니다.

`implements` 함수는 다음과 같이 정의됩니다:

$\text{implements}(\tau, i) = \forall m \in i.\text{Methods}, \exists m' \in \tau.\text{Methods}, m.\text{Signature} = m'.\text{Signature}$

`implements` 함수는 타입 $\tau$가 인터페이스 $i$의 모든 메서드를 구현하는지 확인합니다. 이는 $\tau$의 메서드 집합이 $i$의 메서드 집합의 상위집합인지 검사하는 것과 같습니다.

즉, 인터페이스 $i$의 모든 메서드 $m$에 대해, 타입 $\tau$의 메서드 $m'$가 존재하고, 두 메서드의 시그니처가 동일해야 합니다. 이는 $\tau$의 메서드 집합이 $i$의 메서드 집합을 포함하는지 확인하는 것입니다.

예시:

1. 인터페이스 제약 조건:

```plain
checkConstraint(StringerType, {Interfaces: [Stringer], Types: []}) = true
```

- 여기서 StringerType은 `String() string` 메서드를 가지고 있어야 합니다.

2. 타입 집합 제약 조건:

```plain
checkConstraint(int, {Interfaces: [], Types: [int, float64]}) = true
checkConstraint(string, {Interfaces: [], Types: [int, float64]}) = false
```

- 첫 번째 예시에서 `int`는 주어진 타입 집합에 속하므로 true를 반환합니다.
- 두 번째 예시에서 `string`은 주어진 타입 집합에 속하지 않으므로 false를 반환합니다.

3. 복합 제약 조건:

```plain
checkConstraint(StringerInt, {Interfaces: [Stringer], Types: [int, int64]}) = true
```

- 여기서 StringerInt는 `String() string` 메서드를 가지고 있으며, int 또는 int64 타입이어야 합니다.

4. 다중 인터페이스 제약 조건:

```plain
checkConstraint(ReadWriterCloser, {Interfaces: [Reader, Writer, Closer], Types: []}) = true
```

- 여기서 ReadWriterCloser는 Read, Write, Close 메서드를 모두 구현해야 합니다.
