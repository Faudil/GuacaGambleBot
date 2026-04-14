from enum import Enum
from typing import List, Dict, Any, Optional, Callable
from dataclasses import dataclass, field
from src.utils.i18n import t

class QuestType(Enum):
    MAIN = "main"
    SIDE = "side"
    SEASONAL = "seasonal"
    DAILY = "daily"
    BOUNTY = "bounty"

class QuestStepType(Enum):
    DIALOGUE = "dialogue"   # Story text with "Continue"
    CHOICE = "choice"       # Multiple buttons for branching
    ACTIVITY = "activity"   # Tracking a specific stat (e.g. items_mined)
    REQUIREMENT = "requirement" # Requires money or items
    BOSS_BATTLE = "boss_battle" # PvP-like boss fight with pet
    EVENT = "event"         # Triggered by a specific game event


@dataclass
class QuestReward:
    money: int = 0
    xp: int = 0
    crowns: int = 0
    items: List[str] = field(default_factory=list)
    achievement_id: Optional[str] = None

class QuestStep:
    def __init__(
        self, 
        step_type: QuestStepType, 
        text_key: str, 
        rewards: Optional[QuestReward] = None,
        **kwargs
    ):
        self.step_type = step_type
        self.text_key = text_key
        self.rewards = rewards
        self.extra = kwargs

    def get_text(self, lang: str, **kwargs) -> str:
        return t(self.text_key, lang, **kwargs)

class Quest:
    id: str = ""
    type: QuestType = QuestType.SIDE
    npc_id: Optional[str] = None
    reputation_req: int = 1
    
    def __init__(self):
        self.steps: List[QuestStep] = self.get_steps()

    def get_steps(self) -> List[QuestStep]:
        """Override this to define the quest steps."""
        return []

    def get_title(self, lang: str) -> str:
        return t(f"quests.{self.id}.title", lang)

    def get_description(self, lang: str) -> str:
        return t(f"quests.{self.id}.description", lang)

    def on_activity(self, user_id: int, activity: str, amount: int, current_progress: Dict[str, Any]) -> bool:
        """
        Handle activity updates. 
        Returns True if the current step should be completed.
        """
        return False

    def on_choice(self, user_id: int, choice_id: str, current_progress: Dict[str, Any]) -> int:
        """
        Handle choice selection.
        Returns the next step index.
        """
        return current_progress.get('step_index', 0) + 1

    def get_unlocks(self) -> List[str]:
        """Quests unlocked after completing this one."""
        return []

    def is_available(self, user_id: int) -> bool:
        """Check if this quest can be started by the user."""
        if self.npc_id:
            from src.database.npc import get_reputation
            rep_data = get_reputation(user_id, self.npc_id)
            if rep_data["level"] < self.reputation_req:
                return False
        return True

class QuestRegistry:
    _quests: Dict[str, Quest] = {}

    @classmethod
    def register(cls, quest_class: type):
        quest_instance = quest_class()
        cls._quests[quest_instance.id] = quest_instance
        return quest_class

    @classmethod
    def get_quest(cls, quest_id: str) -> Optional[Quest]:
        return cls._quests.get(quest_id)

    @classmethod
    def get_all_quests(cls) -> List[Quest]:
        return list(cls._quests.values())
