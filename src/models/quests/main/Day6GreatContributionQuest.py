from typing import List, Dict, Any
from src.models.Quest import Quest, QuestStep, QuestStepType, QuestReward

class Day6GreatContributionQuest(Quest):
    @property
    def id(self) -> str:
        return "day6_contribution"

    @property
    def title_key(self) -> str:
        return "quests.day6_contribution.title"

    @property
    def description_key(self) -> str:
        return "quests.day6_contribution.description"

    def get_steps(self) -> List[QuestStep]:
        return [
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.day6_contribution.step0_dialogue",
                rewards=QuestReward(xp=50)
            ),
            QuestStep(
                step_type=QuestStepType.REQUIREMENT,
                text_key="quests.day6_contribution.step1_req",
                rewards=QuestReward(xp=1000),
                requirements={"items": {"minerai de fer": 5, "blé": 5}},
                consume_reqs=True
            )
        ]

    def get_unlocks(self) -> List[str]:
        return ["day7_sprout"]
