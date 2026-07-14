from typing import List, Dict, Any
from src.models.Quest import Quest, QuestStep, QuestStepType, QuestReward

class Day7FirstSproutQuest(Quest):
    @property
    def id(self) -> str:
        return "day7_sprout"

    @property
    def title_key(self) -> str:
        return "quests.day7_sprout.title"

    @property
    def description_key(self) -> str:
        return "quests.day7_sprout.description"

    def get_steps(self) -> List[QuestStep]:
        return [
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.day7_sprout.step0_event",
                rewards=QuestReward(xp=50)
            ),
            QuestStep(
                step_type=QuestStepType.BOSS_BATTLE,
                text_key="quests.day7_sprout.step1_boss",
                rewards=QuestReward(xp=5000, money=2000),
                boss_stats={
                    "name": "Le Gardien Rouillé",
                    "emoji": "🤖",
                    "level": 10,
                    "hp": 200,
                    "atk": 25,
                    "def": 15,
                    "spd": 10,
                    "dge": 2,
                    "acc": 10,
                    "crit_c": 5,
                    "crit_d": 1.5
                }
            ),
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.day7_sprout.step2_dialogue_team",
                rewards=QuestReward(xp=50)
            ),
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.day7_sprout.step3_dialogue_unknown",
                rewards=QuestReward(xp=50)
            )
        ]

    def get_unlocks(self) -> List[str]:
        return []
